package member

import (
	"context"
	"fmt"
	"github.com/StepOnce7/tiflow-operator/api/v1alpha1"
	"github.com/StepOnce7/tiflow-operator/controllers"
	"github.com/StepOnce7/tiflow-operator/controllers/resources"
	"github.com/StepOnce7/tiflow-operator/controllers/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	_ "sigs.k8s.io/controller-runtime/pkg/controller"
)

const (
	// tiflowExecutorDataVolumeMountPath is the mount path for tiflow-executor data volume
	tiflowExecutorDataVolumeMountPath = "/tmp/tiflow-executor"
	// tiflowExecutorClusterVCertPath it where the cert for inter-cluster communication stored (if any)
	tiflowExecutorClusterVCertPath = ""
	// DefaultStorageSize is the default pvc request storage size for dm
	DefaultStorageSize = "10Gi"
)

// ExecutorMemberManager implements interface of Manager.
type ExecutorMemberManager struct {
	Client           client.Client
	ExecutorScale    Scaler
	ExecutorUpgrade  Upgrader
	ExecutorFailover Failover
}

func NewExecutorMemberManager(client client.Client) controllers.Manager {

	// todo: need a new think about how to access the main logic, and what is needed
	// todo: need to implement the logic for Scale, Update, and Failover
	return &ExecutorMemberManager{
		client,
		nil, nil, nil,
	}
}

// Sync implements the logic for syncing tiflowCluster executor member.
func (m *ExecutorMemberManager) Sync(ctx context.Context, tc *v1alpha1.TiflowCluster) error {

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	klog.Info("start to sync tiflowCluster %s:%s ", ns, tcName)

	if tc.Spec.Executor == nil {
		return nil
	}

	// todo: Need to know if master is available？

	// Sync Tiflow Cluster Executor Headless Service
	if err := m.syncExecutorHeadlessService(ctx, tc); err != nil {
		return err
	}

	// Sync Tiflow Cluster Executor StatefulSet
	return m.syncStatefulSet(ctx, tc)
}

// syncExecutorConfigMap implements the logic for syncing configMap of executor.
func (m *ExecutorMemberManager) syncExecutorConfigMap(ctx context.Context, tc *v1alpha1.TiflowCluster, sts *appsv1.StatefulSet) (*corev1.ConfigMap, error) {

	newCfgMap, err := m.getExecutorConfigMap(tc)
	if err != nil {
		return nil, err
	}

	var inUseName string
	if sts != nil {
		inUseName = resources.FindConfigMaoVolume(&sts.Spec.Template.Spec, func(name string) bool {
			return strings.HasPrefix(name, utils.TiFlowExecutorMemberName(tc.Name))
		})
	}
	klog.Info("get executor in use config map name: ", inUseName)

	// todo: Need to finish the UpdateConfigMapIfNeed Logic
	err = resources.UpdateConfigMapIfNeed(ctx, m.Client, v1alpha1.ConfigUpdateStrategyInPlace, inUseName, newCfgMap)
	if err != nil {
		return nil, err
	}

	if err = m.Client.Update(ctx, newCfgMap); err != nil {
		return nil, err
	}

	return newCfgMap, nil
}

// syncExecutorHeadlessService implements the logic for syncing headlessService of executor.
func (m *ExecutorMemberManager) syncExecutorHeadlessService(ctx context.Context, tc *v1alpha1.TiflowCluster) error {

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := m.getNewExecutorHeadlessService(tc)
	oldSvcTmp := &corev1.Service{}
	klog.Info("start to get svc %s.%s", ns, utils.TiFlowExecutorPeerMemberName(tcName))
	err := m.Client.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      utils.TiFlowExecutorPeerMemberName(tcName),
	}, oldSvcTmp)

	klog.Info("get svc %s.%s finished, error: %s, notFound: %v",
		ns, utils.TiFlowExecutorPeerMemberName(tcName), err, errors.IsNotFound(err))

	if errors.IsNotFound(err) {
		err = utils.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}

		return m.Client.Create(ctx, newSvc)
	}

	if err != nil {
		return fmt.Errorf("syncExecutorHeadlessService: failed to get svc %s for cluster %s/%s, error: %s", "executor service",
			ns, utils.TiFlowExecutorPeerMemberName(tcName), err)
	}

	oldSvc := oldSvcTmp.DeepCopy()
	equal, err := utils.ServiceEqual(newSvc, oldSvc)

	if err != nil {
		return err
	}
	if !equal {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		err = utils.SetServiceLastAppliedConfigAnnotation(&svc)
		if err != nil {
			return err
		}

		return m.Client.Update(ctx, newSvc)
	}

	return nil
}

// syncStatefulSet implements the logic for syncing statefulSet of executor.
func (m *ExecutorMemberManager) syncStatefulSet(ctx context.Context, tc *v1alpha1.TiflowCluster) error {

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	klog.Info("start to get sts %s.%s", ns, utils.TiFlowExecutorMemberName(tcName))
	oldStsTmp := &appsv1.StatefulSet{}
	err := m.Client.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      utils.TiFlowExecutorMemberName(tcName),
	}, oldStsTmp)

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncStatefulSet: failed to get sts %s for cluster %s/%s, error: %s ",
			utils.TiFlowExecutorMemberName(tcName), ns, tcName, err)
	}

	stsNotExist := errors.IsNotFound(err)
	oldSts := oldStsTmp.DeepCopy()

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
		// todo: set annotation
		// todo: create new statefulSet
		return nil
	}

	// todo: update new statefulSet
	return resources.UpdateStatefulSetWithPreCheck(tc, "todo", newSts, oldSts)
}

// getExecutorConfigMap returns a new ConfigMap of executor by tiflowCluster Spec.
// Or return a corrected ConfigMap.
func (m *ExecutorMemberManager) getExecutorConfigMap(tc *v1alpha1.TiflowCluster) (*corev1.ConfigMap, error) {

	if tc.Spec.Executor.Config == nil {
		return nil, nil
	}

	config := tc.Spec.Executor.Config.DeepCopy()

	configText, err := config.MarshalTOML()
	if err != nil {
		return nil, err
	}

	startScript, err := utils.RenderExecutorStartScript(&utils.ExecutorStartScriptModel{
		DataDir:       filepath.Join(tiflowExecutorDataVolumeMountPath, tc.Spec.Executor.DataSubDir),
		MasterAddress: utils.TiFLowMasterMemberName(tc.Name) + ":10240",
	})

	if err != nil {
		return nil, err
	}

	instanceName := tc.GetInstanceName()
	executorLabel := utils.NewTiflowCluster().Instance(instanceName).Executor().Labels()

	data := map[string]string{
		"config-file":    string(configText),
		"startup-script": startScript,
	}
	data["config-file"] = string(configText)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            utils.TiFlowExecutorMemberName(tc.Name),
			Namespace:       tc.Namespace,
			Labels:          executorLabel,
			OwnerReferences: []metav1.OwnerReference{utils.GetTiFlowOwnerRef(tc)},
		},
		Data: data,
	}

	return cm, nil
}

// getNewExecutorHeadlessService returns a new headless service of executor by tiflowCluster Spec.
func (m *ExecutorMemberManager) getNewExecutorHeadlessService(tc *v1alpha1.TiflowCluster) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name

	svcName := utils.TiFlowExecutorPeerMemberName(tcName)
	svcLabel := utils.NewTiflowCluster().Instance(tcName).Executor().Labels()

	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: ns,
			Labels:    svcLabel,
			OwnerReferences: []metav1.OwnerReference{
				utils.GetTiFlowOwnerRef(tc),
			},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name:       "worker",
					Port:       10241,
					TargetPort: intstr.FromInt(int(10241)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector:                 svcLabel,
			PublishNotReadyAddresses: true,
		},
	}

	return &svc
}

// getNewExecutorStatefulSet returns a new statefulSet of executor by tiflowCluster Spec.
func (m *ExecutorMemberManager) getNewExecutorStatefulSet(ctx context.Context, tc *v1alpha1.TiflowCluster, cfgMap *corev1.ConfigMap) (*appsv1.StatefulSet, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	// todo: Need to get the baseExecutorSpec
	baseExecutorSpec := tc.BaseExecutorSpec()
	instanceName := tc.GetInstanceName()
	if cfgMap == nil {
		return nil, fmt.Errorf("config-map for tiflow-exeutor is not found, tifloeCluster %s/%s", tc.Namespace, tc.Name)
	}

	executorConfigMap := cfgMap.Name

	annoMout, annoVolume := resources.AnnotationsMountVolume()
	dataVolumeName := string(resources.GetStorageVolumeName("", v1alpha1.TiFlowExecutorMemberType))

	// todo: Need to set MountPath
	volMounts := []corev1.VolumeMount{
		annoMout,
		{Name: "config", ReadOnly: true, MountPath: ""},
		{Name: "startup-script", ReadOnly: true, MountPath: ""},
		{Name: dataVolumeName, ReadOnly: true, MountPath: tiflowExecutorDataVolumeMountPath},
	}

	volMounts = append(volMounts, tc.Spec.Executor.AdditionalVolumeMounts...)

	// todo: will append TLS volume if need

	// todo: Need to modify
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
							Path: "tiflow-executor-start-script.sh",
						},
					},
				},
			},
		},
	}

	// todo: Need to handle the secret if it is existed

	storageSize := DefaultStorageSize
	if tc.Spec.Executor.StorageSize != "" {
		storageSize = tc.Spec.Executor.StorageSize
	}

	rs, err := resource.ParseQuantity(storageSize)
	if err != nil {
		return nil, fmt.Errorf("connot parse storage request for tiflow-executor, tiflowCluster %s/%s, error: %v",
			tc.Namespace,
			tc.Namespace, err)
	}

	storageRequest := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceStorage: rs,
		},
	}

	stsName := utils.TiFlowExecutorMemberName(tcName)
	stsLables := utils.NewTiflowCluster().Instance(instanceName).Executor()
	podLabels := utils.CombineStringMap(stsLables, baseExecutorSpec.Labels())
	// todo: Need to set port
	podAnnotations := utils.CombineStringMap(utils.AnnProm(0), baseExecutorSpec.Annotations())
	stsAnnotations := utils.GetStsAnnotations(tc.Annotations, utils.ExecutorLabelVal)

	executorContainer := corev1.Container{
		Name:            v1alpha1.TiFlowExecutorMemberType.String(),
		Image:           tc.ExecutorImage(),
		ImagePullPolicy: baseExecutorSpec.ImagePullPolicy(),
		Command:         []string{"TODO"},
		Ports: []corev1.ContainerPort{
			{
				Name:          "worker",
				ContainerPort: int32(10240),
				Protocol:      corev1.ProtocolTCP,
			},
		},
		VolumeMounts: volMounts,
		Resources:    resources.ContainerResource(tc.Spec.Executor.ResourceRequirements),
	}

	// todo: Need to modify it
	env := []corev1.EnvVar{
		{
			Name: "NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
	}

	podSpec := baseExecutorSpec.BuildPodSpec()
	if baseExecutorSpec.HostNetwork() {
		// todo: Need to add POD_NAME info
	}

	executorContainer.Env = utils.AppendEnv(env, baseExecutorSpec.Env())
	executorContainer.EnvFrom = baseExecutorSpec.EnvFrom()
	podSpec.Volumes = append(vols, baseExecutorSpec.AdditionalVolumes()...)
	podSpec.Containers = append([]corev1.Container{executorContainer}, baseExecutorSpec.AdditionalContainers()...)
	var initContainers []corev1.Container
	podSpec.InitContainers = append(initContainers, baseExecutorSpec.InitContainers()...)

	executorSts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            stsName,
			Namespace:       ns,
			Labels:          stsLables,
			Annotations:     stsAnnotations,
			OwnerReferences: []metav1.OwnerReference{utils.GetTiFlowOwnerRef(tc)},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(tc.ExecutorStsDesiredReplicas()),
			Selector: stsLables.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: podAnnotations,
				},
				Spec: podSpec,
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: dataVolumeName,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						StorageClassName: tc.Spec.Executor.StorageClassName,
						Resources:        storageRequest,
					},
				},
			},
			ServiceName:         utils.TiFlowExecutorPeerMemberName(tcName),
			PodManagementPolicy: baseExecutorSpec.PodManagementPolicy(),
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: baseExecutorSpec.StatefulSetUpdateStrategy(),
			},
		},
	}

	return executorSts, nil
}

func (m *ExecutorMemberManager) syncExecutorStatus(tc *v1alpha1.TiflowCluster, sts *appsv1.StatefulSet) error {

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