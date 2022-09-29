package member

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/api/label"
	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"github.com/pingcap/tiflow-operator/pkg/component"
	"github.com/pingcap/tiflow-operator/pkg/controller"
	"github.com/pingcap/tiflow-operator/pkg/manager"
	mngerutils "github.com/pingcap/tiflow-operator/pkg/manager/utils"
	"github.com/pingcap/tiflow-operator/pkg/tiflowapi"
	"github.com/pingcap/tiflow-operator/pkg/util"
)

const (
	executorPort = 10241
	// tiflowExecutorDataVolumeMountPath is the mount path for tiflow-executor data volume
	tiflowExecutorDataVolumeMountPath = "/etc/tiflow-executor"
	// tiflowExecutorStorageVolumeMountPath is the mount path for tiflow-executor storage volume
	tiflowExecutorStorageVolumeMountPath = "/mnt/tiflow-executor"
	// tiflowExecutorClusterVCertPath it where the cert for inter-cluster communication stored (if any)
	tiflowExecutorClusterVCertPath = ""
	// DefaultStorageSize is the default pvc request storage size for dm
	DefaultStorageSize = "10Gi"
	// DefaultStorageName is the default pvc name
	DefaultStorageName = "dataflow"
)

// executorMemberManager implements interface of Manager.
type executorMemberManager struct {
	cli       client.Client
	clientSet kubernetes.Interface
	scaler    Scaler
	upgrader  Upgrader
}

func NewExecutorMemberManager(cli client.Client, clientSet kubernetes.Interface) manager.TiflowManager {

	// todo: need to implement the logic for Failover
	return &executorMemberManager{
		cli:       cli,
		clientSet: clientSet,
		scaler:    NewExecutorScaler(clientSet),
		upgrader:  NewExecutorUpgrader(cli),
	}
}

// Sync implements the logic for syncing tiflowCluster executor member.
func (m *executorMemberManager) Sync(ctx context.Context, tc *v1alpha1.TiflowCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	klog.Infof("start to sync tiflow executor [%s/%s]", ns, tcName)

	if tc.Spec.Executor == nil {
		return nil
	}

	if !tc.MasterIsAvailable() {
		return controller.RequeueErrorf("tiflow cluster: %s/%s, waiting for tiflow-master cluster running", ns, tcName)
	}

	// Sync tilfow-Executor Headless Service
	if err := m.syncExecutorHeadlessServiceForTiflowCluster(ctx, tc); err != nil {
		return err
	}

	// Sync tilfow-Executor StatefulSet
	return m.syncExecutorStatefulSetForTiflowCluster(ctx, tc)
}

// syncExecutorConfigMap implements the logic for syncing configMap of executor.
func (m *executorMemberManager) syncExecutorConfigMap(ctx context.Context, tc *v1alpha1.TiflowCluster, sts *appsv1.StatefulSet) (*corev1.ConfigMap, error) {
	newCfgMap, err := m.getExecutorConfigMap(tc)
	if err != nil {
		return nil, err
	}

	var inUseName string
	if sts != nil {
		inUseName = mngerutils.FindConfigMapVolume(&sts.Spec.Template.Spec, func(name string) bool {
			return strings.HasPrefix(name, controller.TiflowExecutorMemberName(tc.Name))
		})
	}

	err = mngerutils.UpdateConfigMapIfNeed(ctx, m.cli, component.BuildExecutorSpec(tc).ConfigUpdateStrategy(), inUseName, newCfgMap)
	if err != nil {
		return nil, err
	}

	result, err := createOrUpdateObject(ctx, m.cli, newCfgMap, mergeConfigMapFunc)
	if err != nil {
		return nil, err
	}
	return result.(*corev1.ConfigMap), nil
}

// syncExecutorHeadlessServiceForTiflowCluster implements the logic for syncing headlessService of executor.
func (m *executorMemberManager) syncExecutorHeadlessServiceForTiflowCluster(ctx context.Context, tc *v1alpha1.TiflowCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := m.getNewExecutorHeadlessService(tc)
	oldSvcTmp := &corev1.Service{}
	err := m.cli.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      controller.TiflowExecutorPeerMemberName(tcName),
	}, oldSvcTmp)
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}

		return m.cli.Create(ctx, newSvc)
	}

	if err != nil {
		return fmt.Errorf("syncExecutorHeadlessService: failed to get svc %s for cluster [%s/%s], error: %s", "executor service",
			ns, controller.TiflowExecutorPeerMemberName(tcName), err)
	}

	oldSvc := oldSvcTmp.DeepCopy()
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

		return m.cli.Update(ctx, newSvc)
	}

	return nil
}

// syncExecutorStatefulSetForTiflowCluster implements the logic for syncing statefulSet of executor.
func (m *executorMemberManager) syncExecutorStatefulSetForTiflowCluster(ctx context.Context, tc *v1alpha1.TiflowCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	oldStsTmp := &appsv1.StatefulSet{}
	err := m.cli.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      controller.TiflowExecutorMemberName(tcName),
	}, oldStsTmp)
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("syncExecutorStatefulSet: failed to get sts %s for cluster [%s/%s], error: %s ",
				controller.TiflowExecutorMemberName(tcName), ns, tcName, err)
		} else {
			// if not, there will be get error: invalid memory address or nil pointer dereference [recovered],
			// when syncExecutorStatus, due to is not found
			oldStsTmp = nil
		}
	}

	stsNotExist := errors.IsNotFound(err)
	oldSts := oldStsTmp.DeepCopy()

	// todo: WIP
	// failed to sync executor status will not affect subsequent logic, just print the errors.
	if err := m.syncExecutorStatus(tc, oldSts); err != nil {
		klog.Errorf("failed to sync TiflowCluster : [%s/%s]'s executor status, error: %v",
			ns, tcName, err)
	}

	// todo: Paused if need, this situation should be handled
	// TOBE

	// Get old configMap if it is existed, and then we will fix it. Instead, we will create a new one.
	cfgMap, err := m.syncExecutorConfigMap(ctx, tc, oldSts)
	if err != nil {
		return err
	}

	// todo: need to handle the failure executor members
	// TOBE

	// Get old statefulSet if it is existed. Instead, we will create a new one.
	newSts, err := m.getNewExecutorStatefulSet(ctx, tc, cfgMap)
	if err != nil {
		return err
	}

	if stsNotExist {
		err = mngerutils.SetStatefulSetLastAppliedConfigAnnotation(newSts)
		if err != nil {
			return err
		}
		if err := m.cli.Create(ctx, newSts); err != nil {
			return err
		}
		tc.Status.Executor.StatefulSet = &appsv1.StatefulSetStatus{}
		return controller.RequeueErrorf("tiflow cluster: [%s/%s], waiting for tiflow-executor cluster running", ns, tcName)
	}

	// Force Update takes precedence over Scaling
	if !tc.Status.Executor.Synced && NeedForceUpgrade(tc.Annotations) {
		tc.Status.Executor.Phase = v1alpha1.UpgradePhase
		mngerutils.SetUpgradePartition(newSts, 0)
		errSts := mngerutils.UpdateStatefulSet(ctx, m.cli, newSts, oldSts)
		return controller.RequeueErrorf("tiflow cluster: [%s/%s]'s tiflow-executor needs force upgrade, %v", ns, tcName, errSts)
	}

	// Scaling takes precedence over normal upgrading because:
	// - if a tiflow-executor fails in the upgrading, users may want to delete it or add
	//   new replicas
	// - it's ok to prune in the middle of upgrading (in statefulset controller
	//   scaling takes precedence over upgrading too)
	if err := m.scaler.Scale(tc, oldSts, newSts); err != nil {
		return err
	}

	if !templateEqual(newSts, oldSts) || tc.Status.Executor.Phase == v1alpha1.UpgradePhase {
		if err := m.upgrader.Upgrade(tc, oldSts, newSts); err != nil {
			return err
		}
	}

	return mngerutils.UpdateStatefulSet(ctx, m.cli, newSts, oldSts)
}

// getExecutorConfigMap returns a new ConfigMap of executor by tiflowCluster Spec.
// Or return a corrected ConfigMap.
func (m *executorMemberManager) getExecutorConfigMap(tc *v1alpha1.TiflowCluster) (*corev1.ConfigMap, error) {
	config := v1alpha1.NewGenericConfig()
	if tc.Spec.Executor.Config != nil {
		config = tc.Spec.Executor.Config.DeepCopy()
	}

	configText, err := config.MarshalTOML()
	if err != nil {
		return nil, err
	}

	// TODO: add discovery or full name to make sure executor can connect to alive master
	masterAddresses := make([]string, 0)
	if !tc.WithoutLocalMaster() {
		masterAddresses = append(masterAddresses, controller.TiflowMasterMemberName(tc.Name)+":10240")
	}
	if tc.Heterogeneous() {
		masterAddresses = append(masterAddresses, controller.TiflowMasterFullHost(tc.Spec.Cluster.Name, tc.Spec.Cluster.Namespace, tc.Spec.ClusterDomain)+":10240") // use dm-master of reference cluster
	}

	startScript, err := RenderExecutorStartScript(&TiflowExecutorStartScriptModel{
		CommonModel: CommonModel{
			ClusterDomain: tc.Spec.ClusterDomain,
		},
		DataDir:       tiflowExecutorDataVolumeMountPath,
		MasterAddress: strings.Join(masterAddresses, ","),
	})
	if err != nil {
		return nil, err
	}

	instanceName := tc.GetInstanceName()
	executorLabel := label.New().Instance(instanceName).TiflowExecutor().Labels()

	data := map[string]string{
		"config-file":    string(configText),
		"startup-script": startScript,
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            controller.TiflowExecutorMemberName(tc.Name),
			Namespace:       tc.Namespace,
			Labels:          executorLabel,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Data: data,
	}

	return cm, nil
}

// getNewExecutorHeadlessService returns a new headless service of executor by tiflowCluster Spec.
func (m *executorMemberManager) getNewExecutorHeadlessService(tc *v1alpha1.TiflowCluster) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	svcName := controller.TiflowExecutorPeerMemberName(tcName)
	instanceName := tc.GetInstanceName()

	executorSelector := label.New().Instance(instanceName).TiflowExecutor()
	svcLabels := executorSelector.Copy().Labels()

	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: ns,
			Labels:    svcLabels,
			OwnerReferences: []metav1.OwnerReference{
				controller.GetOwnerRef(tc),
			},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name:       "tiflow-executor",
					Port:       executorPort,
					TargetPort: intstr.FromInt(executorPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector:                 executorSelector.Labels(),
			PublishNotReadyAddresses: true,
		},
	}

	return &svc
}

// getNewExecutorStatefulSet returns a new statefulSet of executor by tiflowCluster Spec.
func (m *executorMemberManager) getNewExecutorStatefulSet(ctx context.Context, tc *v1alpha1.TiflowCluster, cfgMap *corev1.ConfigMap) (*appsv1.StatefulSet, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	baseExecutorSpec := component.BuildExecutorSpec(tc)
	instanceName := tc.GetInstanceName()
	if cfgMap == nil {
		return nil, fmt.Errorf("config-map for tiflow-exeutor is not found, tifloeCluster [%s/%s]", tc.Namespace, tc.Name)
	}

	stsName := controller.TiflowExecutorMemberName(tcName)
	stsLabels := label.New().Instance(instanceName).TiflowExecutor()

	// can't directly use tc.Annotations here because it will affect tiflowcluster's annotations
	// todo: use getStsAnnotations if we need to use advanced statefulset
	stsAnnotations := map[string]string{}

	podTemp := m.getNewExecutorPodTemp(tc, cfgMap)
	pvcTemp, err := m.getNewExecutorPVCTemp(tc)
	if err != nil {
		return nil, err
	}

	updateStrategy := appsv1.StatefulSetUpdateStrategy{}
	if baseExecutorSpec.StatefulSetUpdateStrategy() == appsv1.OnDeleteStatefulSetStrategyType {
		updateStrategy.Type = appsv1.OnDeleteStatefulSetStrategyType
	} else {
		updateStrategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
		updateStrategy.RollingUpdate = &appsv1.RollingUpdateStatefulSetStrategy{
			Partition: pointer.Int32Ptr(tc.Spec.Executor.Replicas),
		}
	}

	executorSts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            stsName,
			Namespace:       ns,
			Labels:          stsLabels,
			Annotations:     stsAnnotations,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:             pointer.Int32Ptr(tc.ExecutorStsDesiredReplicas()),
			Selector:             stsLabels.LabelSelector(),
			Template:             podTemp,
			VolumeClaimTemplates: pvcTemp,
			ServiceName:          controller.TiflowExecutorPeerMemberName(tcName),
			PodManagementPolicy:  baseExecutorSpec.PodManagementPolicy(),
			UpdateStrategy:       updateStrategy,
		},
	}

	return executorSts, nil
}

// getNewExecutorPodTemp return Pod Temp for Executor StatefulSetSpec
func (m *executorMemberManager) getNewExecutorPodTemp(tc *v1alpha1.TiflowCluster, cfgMap *corev1.ConfigMap) corev1.PodTemplateSpec {

	baseExecutorSpec := component.BuildExecutorSpec(tc)
	podSpec := baseExecutorSpec.BuildPodSpec()

	podVols := m.getNewExecutorPodVols(tc, cfgMap)
	podSpec.Volumes = append(podVols, baseExecutorSpec.AdditionalVolumes()...)

	executorContainer := m.getNewExecutorContainers(tc)
	podSpec.Containers = append(executorContainer, baseExecutorSpec.AdditionalContainers()...)

	var initContainers []corev1.Container
	podSpec.InitContainers = append(initContainers, baseExecutorSpec.InitContainers()...)

	// todo: More information about PodSpec will be modified in the near future

	instanceName := tc.GetInstanceName()
	podLabels := util.CombineStringMap(label.New().Instance(instanceName).TiflowExecutor(), baseExecutorSpec.Labels())
	podAnnotations := util.CombineStringMap(controller.AnnProm(executorPort), baseExecutorSpec.Annotations())

	podTemp := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      podLabels,
			Annotations: podAnnotations,
		},
		Spec: podSpec,
	}

	return podTemp
}

// getNewExecutorPVCTemp getPVC return PVC temp for Executor StatefulSetSpec, used to dynamically create PVs during runtime.
func (m *executorMemberManager) getNewExecutorPVCTemp(tc *v1alpha1.TiflowCluster) ([]corev1.PersistentVolumeClaim, error) {

	storageSize := DefaultStorageSize
	if tc.Spec.Executor.StorageSize != "" {
		storageSize = tc.Spec.Executor.StorageSize
	}

	rs, err := resource.ParseQuantity(storageSize)
	if err != nil {
		return nil, fmt.Errorf("cannot parse storage request for tiflow-executor, tiflowCluster [%s/%s], error: %v",
			tc.Namespace,
			tc.Name, err)
	}

	storageRequest := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceStorage: rs,
		},
	}

	instanceName := tc.GetInstanceName()
	pvcLabels := label.New().Instance(instanceName).TiflowExecutor()
	// pvcAnnotations := tc.Annotations

	// todo: Need to be modified soon
	pvc := []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:            DefaultStorageName,
				Namespace:       tc.GetNamespace(),
				Labels:          pvcLabels,
				OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
				// Annotations: pvcAnnotations,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				StorageClassName: tc.Spec.Executor.StorageClassName,
				Resources:        storageRequest,
			},
		},
	}

	// There are two states of executor in the cluster, one is stateful and the other is stateless.
	// Distinguish between these two states by the label stateful.
	// If it is a stateful executor, set its OwnerReference to delete both its pvc and bound pv when deleting statefulSet.
	// Instead, just delete the statefulSet and keep the pvc and pv.
	// todo: The pvc and pv need to be handled
	// if tc.Spec.Executor.Stateful {
	//	pvc[0].ObjectMeta.Finalizers = []string{}
	// }

	return pvc, nil
}

// getNewExecutorPodVols return Vols for Executor Pod, including anno, config, startup script.
func (m *executorMemberManager) getNewExecutorPodVols(tc *v1alpha1.TiflowCluster, cfgMap *corev1.ConfigMap) []corev1.Volume {
	executorConfigMap := cfgMap.Name
	_, annoVolume := annotationsMountVolume()

	// handle vols for Pod
	vols := []corev1.Volume{
		annoVolume,
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: executorConfigMap,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "config-file",
							Path: "tiflow-executor.toml",
						},
					},
				},
			},
		},
		{
			Name: "startup-script",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: executorConfigMap,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "startup-script",
							Path: "tiflow_executor_start_script.sh",
						},
					},
				},
			},
		},
	}
	if tc.IsClusterTLSEnabled() {
		vols = append(vols, corev1.Volume{
			Name: "tiflow-executor-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterTLSSecretName(tc.Name, label.TiflowMasterLabelVal),
				},
			},
		})
	}

	for _, tlsClientSecretName := range tc.Spec.Executor.TLSClientSecretNames {
		vols = append(vols, corev1.Volume{
			Name: tlsClientSecretName, VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: tlsClientSecretName,
				},
			},
		})
	}

	return vols
}

// getNewExecutorContainers return container for Executor Pod
func (m *executorMemberManager) getNewExecutorContainers(tc *v1alpha1.TiflowCluster) []corev1.Container {
	// handling env infos
	tcName := tc.GetName()
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
			Value: controller.TiflowExecutorPeerMemberName(tcName),
		},
	}

	baseExecutorSpec := component.BuildExecutorSpec(tc)
	if baseExecutorSpec.HostNetwork() {
		env = append(env, corev1.EnvVar{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		})
	}

	env = util.AppendEnv(env, baseExecutorSpec.Env())
	envFrom := baseExecutorSpec.EnvFrom()

	volMounts := m.getNewExecutorContainerVolsMount(tc)

	executorContainer := []corev1.Container{
		{
			Name:            label.TiflowExecutorLabelVal,
			Image:           tc.ExecutorImage(),
			ImagePullPolicy: baseExecutorSpec.ImagePullPolicy(),
			Command:         []string{"/bin/sh", "/usr/local/bin/tiflow_executor_start_script.sh"},
			Ports: []corev1.ContainerPort{
				{
					Name:          "executor",
					ContainerPort: int32(10241),
					Protocol:      corev1.ProtocolTCP,
				},
			},
			Env:          env,
			EnvFrom:      envFrom,
			VolumeMounts: volMounts,
			Resources:    controller.ContainerResource(tc.Spec.Executor.ResourceRequirements),
		},
	}

	return executorContainer
}

// getNewExecutorContainerVolsMount return vols mount info for Executor Container
func (m *executorMemberManager) getNewExecutorContainerVolsMount(tc *v1alpha1.TiflowCluster) []corev1.VolumeMount {
	// add init volume mount, including config and startup-script
	volMounts := []corev1.VolumeMount{
		{Name: "config", ReadOnly: true, MountPath: tiflowExecutorDataVolumeMountPath},
		{Name: "startup-script", ReadOnly: true, MountPath: "/usr/local/bin"},
	}

	if tc.IsClusterTLSEnabled() {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: "tiflow-executor-tls", ReadOnly: true, MountPath: clusterCertPath,
		})
	}

	for _, tlsClientSecretName := range tc.Spec.Executor.TLSClientSecretNames {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: tlsClientSecretName, ReadOnly: true, MountPath: clientCertPath + "/" + tlsClientSecretName,
		})
	}

	// get Annotation mount info, and add it
	annoMount, _ := annotationsMountVolume()
	volMounts = append(volMounts, annoMount)

	// todo: Need to be modified soon
	// handle pvc mount, and add it
	pvcMount := corev1.VolumeMount{
		Name:      DefaultStorageName,
		MountPath: tiflowExecutorStorageVolumeMountPath,
	}
	volMounts = append(volMounts, pvcMount)

	// handling additional mount information for executor
	volMounts = append(volMounts, tc.Spec.Executor.AdditionalVolumeMounts...)

	return volMounts
}

func (m *executorMemberManager) syncExecutorStatus(tc *v1alpha1.TiflowCluster, sts *appsv1.StatefulSet) error {

	// skip if not created yet
	if sts == nil {
		return nil
	}

	// update the status of statefulSet which created by executor in the cluster
	tc.Status.Executor.StatefulSet = &sts.Status

	// todo: How to get Synced info
	upgrading, err := m.executorStatefulSetIsUpgrading(tc, sts)
	if err != nil {
		return err
	}

	if tc.ExecutorStsDesiredReplicas() != *sts.Spec.Replicas {
		tc.Status.Executor.Phase = v1alpha1.ScalePhase
	} else if upgrading {
		tc.Status.Executor.Phase = v1alpha1.UpgradePhase
	} else {
		tc.Status.Executor.Phase = v1alpha1.NormalPhase
	}

	tc.Status.Executor.Members, tc.Status.Executor.PeerMembers, err = m.syncExecutorMembersStatus(tc)
	if err != nil {
		return err
	}
	tc.Status.Executor.Synced = true

	// get follows from podName
	tc.Status.Executor.Image = ""
	if c := m.findContainerByName(sts, "tiflow-executor"); c != nil {
		tc.Status.Executor.Image = c.Image
	}

	// todo: Need to get the info of volumes which running container has bound
	// todo: Waiting for discussion
	// m.syncVolsStatus(tc,sts)
	tc.Status.Executor.Volumes = nil

	return nil
}

func (m *executorMemberManager) executorStatefulSetIsUpgrading(tc *v1alpha1.TiflowCluster, sts *appsv1.StatefulSet) (bool, error) {
	if mngerutils.StatefulSetIsUpgrading(sts) {
		return true, nil
	}

	ns := tc.GetNamespace()
	instanceName := tc.GetInstanceName()
	selector, err := label.New().Instance(instanceName).TiflowExecutor().Selector()
	if err != nil {
		return false, err
	}

	executorPods := &corev1.PodList{}
	err = m.cli.List(context.TODO(), executorPods, client.InNamespace(tc.GetNamespace()), client.MatchingLabelsSelector{Selector: selector})
	if err != nil {
		return false, fmt.Errorf("executorStatefulSetIsupgrading: failed to list pods for cluster [%s/%s], selector %s, error: %v",
			ns, instanceName, selector, err)
	}

	for _, pod := range executorPods.Items {
		revisionHash, exist := pod.Labels[appsv1.ControllerRevisionHashLabelKey]
		if !exist {
			return false, nil
		}
		if revisionHash != tc.Status.Executor.StatefulSet.UpdateRevision {
			return true, nil
		}
	}
	return false, nil
}

func (m *executorMemberManager) findContainerByName(sts *appsv1.StatefulSet, containerName string) *corev1.Container {
	for _, container := range sts.Spec.Template.Spec.Containers {
		if container.Name == containerName {
			return &container
		}
	}
	return nil
}

func (m *executorMemberManager) syncVolsStatus(tc *v1alpha1.TiflowCluster, sts *appsv1.StatefulSet) error {
	// todo:
	selector, err := metav1.LabelSelectorAsSelector(sts.Spec.Selector)
	if err != nil {
		klog.Errorf("converting statefulSet %s selector to metav1 selector", sts.Name)
		return err
	}

	pvs, err := m.clientSet.CoreV1().PersistentVolumes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector.String(),
	})

	for _, pv := range pvs.Items {
		tc.Status.Executor.Volumes[pv.Name] = &v1alpha1.StorageVolumeStatus{
			Name: DefaultStorageName,
			ObservedStorageVolumeStatus: v1alpha1.ObservedStorageVolumeStatus{
				BoundCount:      1,
				CurrentCount:    1,
				ResizedCount:    1,
				CurrentCapacity: resource.Quantity{},
				ResizedCapacity: resource.Quantity{},
			},
		}
	}

	return nil
}

func (m *executorMemberManager) syncExecutorMembersStatus(tc *v1alpha1.TiflowCluster) (map[string]v1alpha1.ExecutorMember, map[string]v1alpha1.ExecutorMember, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if tc.Heterogeneous() && tc.WithoutLocalMaster() {
		ns = tc.Spec.Cluster.Namespace
		tcName = tc.Spec.Cluster.Name
	}

	tiflowClient := tiflowapi.GetMasterClient(m.cli, ns, tcName, "", tc.IsClusterTLSEnabled())

	// get executors info from master
	executorsInfo, err := tiflowClient.GetExecutors()
	if err != nil {
		tc.Status.Executor.Synced = false
		// get endpoints info
		eps := &corev1.Endpoints{}
		epErr := m.cli.Get(context.TODO(), types.NamespacedName{
			Namespace: ns,
			Name:      controller.TiflowMasterMemberName(tcName),
		}, eps)
		if epErr != nil {
			return nil, nil, fmt.Errorf("syncTiflowClusterStatus: failed to get endpoints %s for cluster %s/%s, err: %s, epErr %s",
				controller.TiflowMasterMemberName(tcName), ns, tcName, err, epErr)
		}
		if eps != nil && len(eps.Subsets) == 0 {
			return nil, nil, fmt.Errorf("%s, service %s/%s has no endpoints",
				err, ns, controller.TiflowMasterMemberName(tcName))
		}
		return nil, nil, err
	}

	return syncExecutorMembers(tc, executorsInfo)
}

func syncExecutorMembers(tc *v1alpha1.TiflowCluster, executors tiflowapi.ExecutorsInfo) (map[string]v1alpha1.ExecutorMember, map[string]v1alpha1.ExecutorMember, error) {
	// todo: WIP, get information about the FailureMembers and FailoverUID through the MasterClient
	// sync executors info
	ns := tc.GetNamespace()
	members := make(map[string]v1alpha1.ExecutorMember)
	peerMembers := make(map[string]v1alpha1.ExecutorMember)
	for _, e := range executors.Executors {
		c, err := handleCapability(e.Capability)
		if err != nil {
			return nil, nil, err
		}

		member := v1alpha1.ExecutorMember{
			Id:                 e.ID,
			Name:               e.Name,
			Addr:               e.Address,
			Capability:         c,
			LastTransitionTime: metav1.Now(),
		}
		clusterName, ordinal, namespace, err2 := getOrdinalFromName(e.Name, v1alpha1.TiFlowExecutorMemberType)
		if err2 == nil && clusterName == tc.GetName() && namespace == ns && ordinal < tc.Spec.Master.Replicas {
			members[e.Name] = member
		} else {
			peerMembers[e.Name] = member
		}
	}

	return members, peerMembers, nil
}

func handleCapability(o string) (int64, error) {
	var i interface{}
	d := json.NewDecoder(strings.NewReader(o))
	d.UseNumber()

	if err := d.Decode(&i); err != nil {
		return -1, err
	}

	n := i.(json.Number)
	res, err := n.Int64()
	if err != nil {
		return -1, err
	}

	return res, nil
}

func formatExecutorName(name string) (string, error) {
	nameSlice := strings.Split(name, ".")
	if len(nameSlice) != 4 {
		return "", fmt.Errorf("split name %s error", name)
	}

	res := fmt.Sprintf("%s.%s", nameSlice[0], nameSlice[2])
	return res, nil
}
