package member

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/api/label"
	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"github.com/pingcap/tiflow-operator/pkg/component"
	"github.com/pingcap/tiflow-operator/pkg/condition"
	"github.com/pingcap/tiflow-operator/pkg/controller"
	"github.com/pingcap/tiflow-operator/pkg/manager"
	mngerutils "github.com/pingcap/tiflow-operator/pkg/manager/utils"
	"github.com/pingcap/tiflow-operator/pkg/status"
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

	// if !tc.MasterIsAvailable() {
	// 	return controller.RequeueErrorf("tiflow cluster: %s/%s, waiting for tiflow-master cluster running", ns, tcName)
	// }

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
		condition.SetFalse(v1alpha1.ExecutorSyncChecked, tc.GetClusterStatus(), metav1.Now())
		status.Ongoing(v1alpha1.CreateType, tc.GetClusterStatus(), v1alpha1.TiFlowExecutorMemberType,
			fmt.Sprintf("start to create executor cluster [%s/%s]", ns, tcName))

		err = mngerutils.SetStatefulSetLastAppliedConfigAnnotation(newSts)
		if err != nil {
			return err
		}

		if err := m.cli.Create(ctx, newSts); err != nil {
			status.Failed(v1alpha1.CreateType, tc.GetClusterStatus(), v1alpha1.TiFlowExecutorMemberType,
				fmt.Sprintf("create executor cluster [%s/%s] failed", ns, tcName))
			return err
		}

		tc.Status.Executor.StatefulSet = &appsv1.StatefulSetStatus{}
		return nil
	}

	if condition.False(v1alpha1.ExecutorSyncChecked, tc.GetClusterConditions()) {
		// Force Update takes precedence over Scaling
		if NeedForceUpgrade(tc.Annotations) {
			tc.Status.Executor.Phase = v1alpha1.ExecutorUpgrading
			mngerutils.SetUpgradePartition(newSts, 0)
			errSts := mngerutils.UpdateStatefulSet(ctx, m.cli, newSts, oldSts)
			return controller.RequeueErrorf("tiflow cluster: [%s/%s]'s tiflow-executor needs force upgrade, %v", ns, tcName, errSts)
		}

		klog.Info("waiting for tiflow-executor's status sync")
		return nil
	}

	// Scaling takes precedence over normal upgrading because:
	// - if a tiflow-executor fails in the upgrading, users may want to delete it or add
	//   new replicas
	// - it's ok to prune in the middle of upgrading (in statefulset controller
	//   scaling takes precedence over upgrading too)
	if err := m.scaler.Scale(tc, oldSts, newSts); err != nil {
		return err
	}

	if !templateEqual(newSts, oldSts) || tc.Status.Executor.Phase == v1alpha1.ExecutorUpgrading {
		if err := m.upgrader.Upgrade(tc, oldSts, newSts); err != nil {
			status.Failed(v1alpha1.UpgradeType, tc.GetClusterStatus(), v1alpha1.TiFlowExecutorMemberType,
				fmt.Sprintf("tiflow executor [%s/%s] upgrading failed", ns, tcName))
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
			Replicas:            pointer.Int32Ptr(tc.ExecutorStsDesiredReplicas()),
			Selector:            stsLabels.LabelSelector(),
			Template:            podTemp,
			ServiceName:         controller.TiflowExecutorPeerMemberName(tcName),
			PodManagementPolicy: baseExecutorSpec.PodManagementPolicy(),
			UpdateStrategy:      updateStrategy,
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

	// handling additional mount information for executor
	volMounts = append(volMounts, tc.Spec.Executor.AdditionalVolumeMounts...)

	return volMounts
}

// func (m *executorMemberManager) syncVolsStatus(tc *v1alpha1.TiflowCluster, sts *appsv1.StatefulSet) error {
// 	// todo:
// 	selector, err := metav1.LabelSelectorAsSelector(sts.Spec.Selector)
// 	if err != nil {
// 		klog.Errorf("converting statefulSet %s selector to metav1 selector", sts.Name)
// 		return err
// 	}
//
// 	pvs, err := m.clientSet.CoreV1().PersistentVolumes().List(context.TODO(), metav1.ListOptions{
// 		LabelSelector: selector.String(),
// 	})
//
// 	for _, pv := range pvs.Items {
// 		tc.Status.Executor.Volumes[pv.Name] = &v1alpha1.StorageVolumeStatus{
// 			Name: DefaultStorageName,
// 			ObservedStorageVolumeStatus: v1alpha1.ObservedStorageVolumeStatus{
// 				BoundCount:      1,
// 				CurrentCount:    1,
// 				ResizedCount:    1,
// 				CurrentCapacity: resource.Quantity{},
// 				ResizedCapacity: resource.Quantity{},
// 			},
// 		}
// 	}
//
// 	return nil
// }
