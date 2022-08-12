package member

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/resource"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"github.com/pingcap/tiflow-operator/pkg/component"
	"github.com/pingcap/tiflow-operator/pkg/controller"
	"github.com/pingcap/tiflow-operator/pkg/label"
	"github.com/pingcap/tiflow-operator/pkg/manager"
	mngerutils "github.com/pingcap/tiflow-operator/pkg/manager/utils"
	"github.com/pingcap/tiflow-operator/pkg/util"
)

const (
	executorPort = 10241
	// tiflowExecutorDataVolumeMountPath is the mount path for tiflow-executor data volume
	tiflowExecutorDataVolumeMountPath = "/etc/tiflow-executor"
	// tiflowExecutorStorageVolumeMountPath is the mount path for tiflow-executor storage volume
	tiflowExecutorStorageVolumeMountPath = "/tmp/tiflow-executor"
	// tiflowExecutorClusterVCertPath it where the cert for inter-cluster communication stored (if any)
	tiflowExecutorClusterVCertPath = ""
	// DefaultStorageSize is the default pvc request storage size for dm
	DefaultStorageSize = "10Gi"
	// DefaultStorageName is the default pvc name
	DefaultStorageName = "pvc"
)

// executorMemberManager implements interface of Manager.
type executorMemberManager struct {
	Client   client.Client
	Scale    Scaler
	Upgrade  Upgrader
	Failover Failover
}

func NewExecutorMemberManager(client client.Client) manager.TiflowManager {

	// todo: need a new think about how to access the main logic, and what is needed
	// todo: need to implement the logic for Scale, Update, and Failover
	return &executorMemberManager{
		client,
		nil, nil, nil,
	}
}

// Sync implements the logic for syncing tiflowCluster executor member.
func (m *executorMemberManager) Sync(ctx context.Context, tc *v1alpha1.TiflowCluster) error {

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	klog.Infof("start to sync tiflow cluster [%s/%s]", ns, tcName)

	if tc.Spec.Executor == nil {
		return nil
	}

	// todo: Need to know if master is available？

	// Sync Tiflow Cluster Executor Headless Service
	if err := m.syncExecutorHeadlessServiceForTiflowCluster(ctx, tc); err != nil {
		return err
	}

	// Sync Tiflow Cluster Executor StatefulSet
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
	klog.Infof("get executor in use config map name: %s", inUseName)

	// todo: Need to finish the UpdateConfigMapIfNeed Logic
	err = mngerutils.UpdateConfigMapIfNeed(ctx, m.Client, component.BuildExecutorSpec(tc).ConfigUpdateStrategy(), inUseName, newCfgMap)
	if err != nil {
		return nil, err
	}

	//if err = m.Client.Update(ctx, newCfgMap); err != nil {
	//	return nil, err
	//}

	result, err := createOrUpdateObject(ctx, m.Client, newCfgMap, mergeConfigMapFunc)
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
	klog.Infof("start to get svc %s.%s", ns, controller.TiflowExecutorPeerMemberName(tcName))
	err := m.Client.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      controller.TiflowExecutorPeerMemberName(tcName),
	}, oldSvcTmp)

	klog.Infof("get svc %s.%s finished, error: %v, notFound: %v",
		ns, controller.TiflowExecutorPeerMemberName(tcName), err, errors.IsNotFound(err))

	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}

		return m.Client.Create(ctx, newSvc)
	}

	if err != nil {
		return fmt.Errorf("syncExecutorHeadlessService: failed to get svc %s for cluster %s/%s, error: %s", "executor service",
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

		return m.Client.Update(ctx, newSvc)
	}

	return nil
}

// syncExecutorStatefulSetForTiflowCluster implements the logic for syncing statefulSet of executor.
func (m *executorMemberManager) syncExecutorStatefulSetForTiflowCluster(ctx context.Context, tc *v1alpha1.TiflowCluster) error {

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	klog.Infof("start to get sts %s.%s", ns, controller.TiflowExecutorMemberName(tcName))
	oldStsTmp := &appsv1.StatefulSet{}
	err := m.Client.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      controller.TiflowExecutorMemberName(tcName),
	}, oldStsTmp)

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncStatefulSet: failed to get sts %s for cluster %s/%s, error: %s ",
			controller.TiflowExecutorMemberName(tcName), ns, tcName, err)
	}

	stsNotExist := errors.IsNotFound(err)
	oldSts := oldStsTmp.DeepCopy()

	// failed to sync executor status will not affect subsequent logic, just print the errors.
	// todo:
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
		if err := m.Client.Create(ctx, newSts); err != nil {
			return err
		}
		tc.Status.Executor.StatefulSet = &appsv1.StatefulSetStatus{}
		return controller.RequeueErrorf("TiflowCluster: [%s/%s], waiting for tiflow-executor cluster running", ns, tcName)
	}

	return mngerutils.UpdateStatefulSet(ctx, m.Client, newSts, oldSts)
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

	startScript, err := RenderExecutorStartScript(&TiflowExecutorStartScriptModel{
		DataDir:       tiflowExecutorDataVolumeMountPath,
		MasterAddress: controller.TiflowMasterMemberName(tc.Name) + ":10240",
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
	svcLabels := executorSelector.Copy().UsedByPeer().Labels()

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
		return nil, fmt.Errorf("config-map for tiflow-exeutor is not found, tifloeCluster %s/%s", tc.Namespace, tc.Name)
	}

	// todo: Need to handle the secret if it is existed

	stsName := controller.TiflowExecutorMemberName(tcName)
	stsLabels := label.New().Instance(instanceName).TiflowExecutor()
	stsAnnotations := tc.Annotations

	podTemp := m.getNewExecutorPodTemp(tc, cfgMap)
	pvcTemp, err := m.getNewExecutorPVCTemp(tc)
	if err != nil {
		return nil, err
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
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: baseExecutorSpec.StatefulSetUpdateStrategy(),
			},
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
		return nil, fmt.Errorf("connot parse storage request for tiflow-executor, tiflowCluster %s/%s, error: %v",
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
	//pvcAnnotations := tc.Annotations

	// todo: Need to be modified soon
	pvc := []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      DefaultStorageName,
				Namespace: tc.GetNamespace(),
				Labels:    pvcLabels,
				//Annotations: pvcAnnotations,
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
	// todo: The pvc and pv need to be handled separately
	if tc.Spec.Executor.Stateful {
		pvc[0].ObjectMeta.OwnerReferences = []metav1.OwnerReference{controller.GetOwnerRef(tc)}
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

	// nn old statefulSet exists
	//clusterNameSpace := cluster.GetNamespace()
	//clusterName := cluster.GetName()

	// update the status of statefulSet which created by executor in the cluster
	tc.Status.Executor.StatefulSet = &sts.Status

	// todo: How to get Synced info
	// todo: Need to check if the current sts are updating
	tc.Status.Executor.Phase = v1alpha1.NormalPhase

	// todo: Get information about the Executor Members, FailureMembers and FailoverUID through the Master API
	// todo: Or may be get info through the Sts Status
	// TOBE

	// todo: Need to get the info of the running version image in the cluster
	tc.Status.Executor.Image = ""

	// todo: Need to get the info of volumes which running container has bound
	tc.Status.Executor.Volumes = nil
	return nil
}

func loadExecutorInitContainerCommand(tc *v1alpha1.TiflowCluster) []string {

	masterSvcName := controller.TiflowMasterMemberName(tc.Name)

	listenStr := masterSvcName + "-0" + "." + masterSvcName

	origin := `until nslookup tmp;
do
	echo waiting for tmp;
	sleep 2;
done;`

	currentStr := strings.Replace(origin, "tmp", listenStr, -1)

	return []string{
		"sh", "-c",
		currentStr,
	}
}
