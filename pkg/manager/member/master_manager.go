package member

import (
	"context"
	"fmt"
	"strings"

	pingcapcomv1alpha1 "github.com/StepOnce7/tiflow-operator/api/v1alpha1"
	"github.com/StepOnce7/tiflow-operator/pkg/component"
	"github.com/StepOnce7/tiflow-operator/pkg/controller"
	"github.com/StepOnce7/tiflow-operator/pkg/label"
	"github.com/StepOnce7/tiflow-operator/pkg/manager"
	mngerutils "github.com/StepOnce7/tiflow-operator/pkg/manager/utils"
	"github.com/StepOnce7/tiflow-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	masterPort     = 10240
	masterPeerPort = 8291
)

type masterMemberManager struct {
	client.Client
}

func NewMasterMemberManager(cli client.Client) manager.TiflowManager {
	return &masterMemberManager{cli}
}

func (m *masterMemberManager) Sync(ctx context.Context, tc *pingcapcomv1alpha1.TiflowCluster) error {
	if tc.Spec.Master == nil {
		return nil
	}
	// Sync tiflow-master Service
	if err := m.syncMasterServiceForTiflowCluster(ctx, tc); err != nil {
		return err
	}

	// Sync tiflow-master Headless Service
	if err := m.syncMasterHeadlessServiceForTiflowCluster(ctx, tc); err != nil {
		return err
	}

	// Sync tiflow-master StatefulSet
	return m.syncMasterStatefulSetForTiflowCluster(ctx, tc)
}

func getMasterConfigMap(tc *pingcapcomv1alpha1.TiflowCluster) (*corev1.ConfigMap, error) {
	config := pingcapcomv1alpha1.NewGenericConfig()
	if tc.Spec.Master.Config != nil {
		config = tc.Spec.Master.Config.DeepCopy()
	}

	// TODO: tls
	// override CA if tls enabled
	//if tc.IsTLSClusterEnabled() {
	//	config.Set("ssl-ca", path.Join(tiflowMasterClusterCertPath, tlsSecretRootCAKey))
	//	config.Set("ssl-cert", path.Join(tiflowMasterClusterCertPath, corev1.TLSCertKey))
	//	config.Set("ssl-key", path.Join(tiflowMasterClusterCertPath, corev1.TLSPrivateKeyKey))
	//}

	confText, err := config.MarshalTOML()
	if err != nil {
		return nil, err
	}

	startScript, err := RenderTiflowMasterStartScript(&TiflowMasterStartScriptModel{
		Scheme: tc.Scheme(),
	})
	if err != nil {
		return nil, err
	}

	instanceName := tc.GetInstanceName()
	masterLabel := label.New().Instance(instanceName).TiflowMaster().Labels()
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            controller.TiflowMasterMemberName(tc.Name),
			Namespace:       tc.Namespace,
			Labels:          masterLabel,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Data: map[string]string{
			"config-file":    string(confText),
			"startup-script": startScript,
		},
	}
	return cm, nil
}

// syncMasterConfigMap syncs the configmap of tiflow-master
func (m *masterMemberManager) syncMasterConfigMap(ctx context.Context, tc *pingcapcomv1alpha1.TiflowCluster, set *apps.StatefulSet) (*corev1.ConfigMap, error) {
	newCm, err := getMasterConfigMap(tc)
	if err != nil {
		return nil, err
	}

	var inUseName string
	if set != nil {
		inUseName = mngerutils.FindConfigMapVolume(&set.Spec.Template.Spec, func(name string) bool {
			return strings.HasPrefix(name, controller.TiflowMasterMemberName(tc.Name))
		})
	}

	err = mngerutils.UpdateConfigMapIfNeed(ctx, m.Client, component.BuildMasterSpec(tc).ConfigUpdateStrategy(), inUseName, newCm)
	if err != nil {
		return nil, err
	}
	result, err := createOrUpdateObject(ctx, m.Client, newCm, mergeConfigMapFunc)
	if err != nil {
		return nil, err
	}
	return result.(*corev1.ConfigMap), nil
}

func (m *masterMemberManager) syncMasterServiceForTiflowCluster(ctx context.Context, tc *pingcapcomv1alpha1.TiflowCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := m.getNewMasterServiceForTiflowCluster(tc)
	oldSvcTmp := &corev1.Service{}
	klog.Infof("start to get svc %s.%s", ns, controller.TiflowMasterMemberName(tcName))
	err := m.Client.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      controller.TiflowMasterMemberName(tcName),
	}, oldSvcTmp)
	klog.Infof("get svc %s.%s finish, error: %s, notFound: %v, content: %s", ns, controller.TiflowMasterMemberName(tcName), err, errors.IsNotFound(err), oldSvcTmp)
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return m.Client.Create(ctx, newSvc)
	}
	if err != nil {
		return fmt.Errorf("syncMasterServiceForTiflowCluster: failed to get svc %s for cluster %s/%s, error: %s", controller.TiflowMasterMemberName(tcName), ns, tcName, err)
	}

	oldSvc := oldSvcTmp.DeepCopy()
	util.RetainManagedFields(newSvc, oldSvc)

	equal, err := controller.ServiceEqual(newSvc, oldSvc)
	klog.Infof("check svc result, equal: %v, error: %s, newSvc: %s", equal, err, newSvc)
	if err != nil {
		return err
	}
	if !equal {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		err = controller.SetServiceLastAppliedConfigAnnotation(&svc)
		if err != nil {
			return err
		}
		svc.Spec.ClusterIP = oldSvc.Spec.ClusterIP
		for k, v := range newSvc.Annotations {
			svc.Annotations[k] = v
		}
		err = m.Client.Update(ctx, &svc)
		return err
	}

	return nil
}

func (m *masterMemberManager) syncMasterHeadlessServiceForTiflowCluster(ctx context.Context, tc *pingcapcomv1alpha1.TiflowCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := getNewMasterHeadlessServiceForTiflowCluster(tc)
	oldSvc := &corev1.Service{}
	err := m.Client.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      controller.TiflowMasterPeerMemberName(tcName),
	}, oldSvc)
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return m.Client.Create(ctx, newSvc)
	}
	if err != nil {
		return fmt.Errorf("syncMasterHeadlessServiceForTiflowCluster: failed to get svc %s for cluster %s/%s, error: %s", controller.TiflowMasterPeerMemberName(tcName), ns, tcName, err)
	}

	equal, err := controller.ServiceEqual(newSvc, oldSvc)
	if err != nil {
		return err
	}
	if !equal {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		err = controller.SetServiceLastAppliedConfigAnnotation(&svc)
		if err != nil {
			return err
		}
		return m.Client.Update(ctx, &svc)
	}

	return nil
}

func (m *masterMemberManager) syncMasterStatefulSetForTiflowCluster(ctx context.Context, tc *pingcapcomv1alpha1.TiflowCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	oldMasterSetTmp := &apps.StatefulSet{}
	err := m.Client.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      controller.TiflowMasterMemberName(tcName),
	}, oldMasterSetTmp)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncMasterStatefulSetForTiflowCluster: fail to get sts %s for cluster %s/%s, error: %s", controller.TiflowMasterMemberName(tcName), ns, tcName, err)
	}

	setNotExist := errors.IsNotFound(err)
	oldMasterSet := oldMasterSetTmp.DeepCopy()

	if err = m.syncTiflowClusterStatus(tc, oldMasterSet); err != nil {
		klog.Errorf("failed to sync DMCluster: [%s/%s]'s status, error: %v", ns, tcName, err)
	}

	cm, err := m.syncMasterConfigMap(ctx, tc, oldMasterSet)
	if err != nil {
		return err
	}
	newMasterSet, err := getNewMasterSetForTiflowCluster(tc, cm)
	if err != nil {
		return err
	}
	if setNotExist {
		err = mngerutils.SetStatefulSetLastAppliedConfigAnnotation(newMasterSet)
		if err != nil {
			return err
		}
		if err := m.Client.Create(ctx, newMasterSet); err != nil {
			return err
		}
		tc.Status.Master.StatefulSet = &apps.StatefulSetStatus{}
		return controller.RequeueErrorf("TiflowCluster: [%s/%s], waiting for tiflow-master cluster running", ns, tcName)
	}

	// Force update takes precedence over scaling because force upgrade won't take effect when cluster gets stuck at scaling
	if !tc.Status.Master.Synced && NeedForceUpgrade(tc.Annotations) {
		tc.Status.Master.Phase = pingcapcomv1alpha1.UpgradePhase
		mngerutils.SetUpgradePartition(newMasterSet, 0)
		errSTS := mngerutils.UpdateStatefulSet(ctx, m.Client, newMasterSet, oldMasterSet)
		return controller.RequeueErrorf("tiflowcluster: [%s/%s]'s tiflow-master needs force upgrade, %v", ns, tcName, errSTS)
	}

	// TODO: support scaler
	// Scaling takes precedence over normal upgrading because:
	// - if a tiflow-master fails in the upgrading, users may want to delete it or add
	//   new replicas
	// - it's ok to scale in the middle of upgrading (in statefulset controller
	//   scaling takes precedence over upgrading too)
	//if err := m.scaler.Scale(tc, oldMasterSet, newMasterSet); err != nil {
	//	return err
	//}

	// TODO: support auto failover
	// Perform failover logic if necessary. Note that this will only update
	// DMCluster status. The actual scaling performs in next sync loop (if a
	// new replica needs to be added).
	//if m.deps.CLIConfig.AutoFailover {
	//	if m.shouldRecover(tc) {
	//		m.failover.Recover(tc)
	//	} else if tc.MasterAllPodsStarted() && !tc.MasterAllMembersReady() || tc.MasterAutoFailovering() {
	//		if err := m.failover.Failover(tc); err != nil {
	//			return err
	//		}
	//	}
	//}

	// TODO: support upgrader
	//if !templateEqual(newMasterSet, oldMasterSet) || tc.Status.Master.Phase == pingcapcomv1alpha1.UpgradePhase {
	//	if err := m.upgrader.Upgrade(tc, oldMasterSet, newMasterSet); err != nil {
	//		return err
	//	}
	//}

	return mngerutils.UpdateStatefulSet(ctx, m.Client, newMasterSet, oldMasterSet)
}

func getNewMasterSetForTiflowCluster(tc *pingcapcomv1alpha1.TiflowCluster, cm *corev1.ConfigMap) (*apps.StatefulSet, error) {
	ns := tc.Namespace
	tcName := tc.Name
	baseMasterSpec := component.BuildMasterSpec(tc)
	instanceName := tc.GetInstanceName()
	if cm == nil {
		return nil, fmt.Errorf("config map for tiflow-master is not found, dmcluster %s/%s", tc.Namespace, tc.Name)
	}
	masterConfigMap := cm.Name

	annoMount, annoVolume := annotationsMountVolume()
	volMounts := []corev1.VolumeMount{
		annoMount,
		{Name: "config", ReadOnly: true, MountPath: "/etc/tiflow-master"},
		{Name: "startup-script", ReadOnly: true, MountPath: "/usr/local/bin"},
	}
	volMounts = append(volMounts, tc.Spec.Master.AdditionalVolumeMounts...)

	// TODO: tls
	//if tc.IsTLSClusterEnabled() {
	//	volMounts = append(volMounts, corev1.VolumeMount{
	//		Name: "tiflow-master-tls", ReadOnly: true, MountPath: "/var/lib/tiflow-master-tls",
	//	})
	//}

	vols := []corev1.Volume{
		annoVolume,
		{Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: masterConfigMap,
					},
					Items: []corev1.KeyToPath{{Key: "config-file", Path: "tiflow-master.toml"}},
				},
			},
		},
		{Name: "startup-script",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: masterConfigMap,
					},
					Items: []corev1.KeyToPath{{Key: "startup-script", Path: "dm_master_start_script.sh"}},
				},
			},
		},
	}
	// TODO: tls
	//if tc.IsTLSClusterEnabled() {
	//	vols = append(vols, corev1.Volume{
	//		Name: "tiflow-master-tls", VolumeSource: corev1.VolumeSource{
	//			Secret: &corev1.SecretVolumeSource{
	//				SecretName: util.ClusterTLSSecretName(tc.Name, label.TiflowMaster),
	//			},
	//		},
	//	})
	//}
	//
	//for _, tlsClientSecretName := range tc.Spec.TLSClientSecretNames {
	//	volMounts = append(volMounts, corev1.VolumeMount{
	//		Name: tlsClientSecretName, ReadOnly: true, MountPath: fmt.Sprintf("/var/lib/source-tls/%s", tlsClientSecretName),
	//	})
	//	vols = append(vols, corev1.Volume{
	//		Name: tlsClientSecretName, VolumeSource: corev1.VolumeSource{
	//			Secret: &corev1.SecretVolumeSource{
	//				SecretName: tlsClientSecretName,
	//			},
	//		},
	//	})
	//}

	setName := controller.TiflowMasterMemberName(tcName)
	stsLabels := label.New().Instance(instanceName).TiflowMaster()
	stsAnnotations := tc.Annotations
	podLabels := util.CombineStringMap(stsLabels, baseMasterSpec.Labels())
	podAnnotations := util.CombineStringMap(controller.AnnProm(masterPort), baseMasterSpec.Annotations())
	// TODO: support failover
	//failureReplicas := getTiflowMasterFailureReplicas(tc)

	masterContainer := corev1.Container{
		Name:            label.TiflowMasterLabelVal,
		Image:           tc.MasterImage(),
		ImagePullPolicy: baseMasterSpec.ImagePullPolicy(),
		Command:         []string{"/bin/sh", "/usr/local/bin/dm_master_start_script.sh"},
		Ports: []corev1.ContainerPort{
			{
				Name:          "peer",
				ContainerPort: int32(8291),
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "client",
				ContainerPort: int32(8261),
				Protocol:      corev1.ProtocolTCP,
			},
		},
		VolumeMounts: volMounts,
		Resources:    controller.ContainerResource(tc.Spec.Master.ResourceRequirements),
	}
	env := []corev1.EnvVar{
		{
			Name: "NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name:  "PEER_SERVICE_NAME",
			Value: controller.TiflowMasterPeerMemberName(tcName),
		},
		{
			Name:  "SERVICE_NAME",
			Value: controller.TiflowMasterMemberName(tcName),
		},
		{
			Name:  "SET_NAME",
			Value: setName,
		},
	}

	podSpec := baseMasterSpec.BuildPodSpec()
	if baseMasterSpec.HostNetwork() {
		env = append(env, corev1.EnvVar{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		})
	}
	masterContainer.Env = util.AppendEnv(env, baseMasterSpec.Env())
	masterContainer.EnvFrom = baseMasterSpec.EnvFrom()
	podSpec.Volumes = append(vols, baseMasterSpec.AdditionalVolumes()...)
	podSpec.Containers = append([]corev1.Container{masterContainer}, baseMasterSpec.AdditionalContainers()...)
	var initContainers []corev1.Container // no default initContainers now
	podSpec.InitContainers = append(initContainers, baseMasterSpec.InitContainers()...)

	updateStrategy := apps.StatefulSetUpdateStrategy{}
	if baseMasterSpec.StatefulSetUpdateStrategy() == apps.OnDeleteStatefulSetStrategyType {
		updateStrategy.Type = apps.OnDeleteStatefulSetStrategyType
	} else {
		updateStrategy.Type = apps.RollingUpdateStatefulSetStrategyType
		updateStrategy.RollingUpdate = &apps.RollingUpdateStatefulSetStrategy{
			Partition: pointer.Int32Ptr(tc.Spec.Master.Replicas),
			//TODO: support failover later
			//Partition: pointer.Int32Ptr(tc.Spec.Master.Replicas + int32(failureReplicas)),
		}
	}

	masterSet := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            setName,
			Namespace:       ns,
			Labels:          stsLabels,
			Annotations:     stsAnnotations,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(tc.Spec.Master.Replicas),
			//TODO: support failover later
			//Replicas: pointer.Int32Ptr(tc.Spec.Master.Replicas + int32(failureReplicas)),
			Selector: stsLabels.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: podAnnotations,
				},
				Spec: podSpec,
			},
			ServiceName:         controller.TiflowMasterPeerMemberName(tcName),
			PodManagementPolicy: baseMasterSpec.PodManagementPolicy(),
			UpdateStrategy:      updateStrategy,
		},
	}

	return masterSet, nil
}

func (m *masterMemberManager) getNewMasterServiceForTiflowCluster(tc *pingcapcomv1alpha1.TiflowCluster) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	svcName := controller.TiflowMasterMemberName(tcName)
	instanceName := tc.GetInstanceName()
	masterSelector := label.New().Instance(instanceName).TiflowMaster()
	masterLabels := masterSelector.Copy().UsedByEndUser().Labels()

	ports := []corev1.ServicePort{
		{
			Name:       "tiflow-master",
			Port:       masterPort,
			TargetPort: intstr.FromInt(masterPort),
			Protocol:   corev1.ProtocolTCP,
		},
	}
	masterSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          masterLabels,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Ports:    ports,
			Selector: masterSelector.Labels(),
		},
	}

	// override fields with user-defined ServiceSpec
	svcSpec := tc.Spec.Master.Service
	if svcSpec != nil {
		if svcSpec.Type != "" {
			masterSvc.Spec.Type = svcSpec.Type
		}
		masterSvc.ObjectMeta.Annotations = util.CopyStringMap(svcSpec.Annotations)
		masterSvc.ObjectMeta.Labels = util.CombineStringMap(masterSvc.ObjectMeta.Labels, svcSpec.Labels)
		masterSvc.Spec.Ports[0].NodePort = getNodePort(svcSpec)
		if svcSpec.Type == corev1.ServiceTypeLoadBalancer {
			if svcSpec.LoadBalancerIP != nil {
				masterSvc.Spec.LoadBalancerIP = *svcSpec.LoadBalancerIP
			}
			if svcSpec.LoadBalancerSourceRanges != nil {
				masterSvc.Spec.LoadBalancerSourceRanges = svcSpec.LoadBalancerSourceRanges
			}
		}
		if svcSpec.ExternalTrafficPolicy != nil {
			masterSvc.Spec.ExternalTrafficPolicy = *svcSpec.ExternalTrafficPolicy
		}
		if svcSpec.ClusterIP != nil {
			masterSvc.Spec.ClusterIP = *svcSpec.ClusterIP
		}
		if svcSpec.PortName != nil {
			masterSvc.Spec.Ports[0].Name = *svcSpec.PortName
		}
	}
	return masterSvc
}

func getNewMasterHeadlessServiceForTiflowCluster(tc *pingcapcomv1alpha1.TiflowCluster) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	svcName := controller.TiflowMasterPeerMemberName(tcName)
	instanceName := tc.GetInstanceName()
	masterSelector := label.New().Instance(instanceName).TiflowMaster()
	masterLabels := masterSelector.Copy().UsedByPeer().Labels()

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          masterLabels,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name:       "tiflow-master-peer",
					Port:       masterPeerPort,
					TargetPort: intstr.FromInt(masterPeerPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector:                 masterSelector.Labels(),
			PublishNotReadyAddresses: true,
		},
	}
}

func (m *masterMemberManager) syncTiflowClusterStatus(tc *pingcapcomv1alpha1.TiflowCluster, set *apps.StatefulSet) error {
	if set == nil {
		// skip if not created yet
		return nil
	}
	// TODO: add status info after tiflow master interface stable
	return nil
}
