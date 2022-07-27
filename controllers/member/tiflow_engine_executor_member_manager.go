package member

import (
	"fmt"
	"github.com/StepOnce7/tiflow-operator/api/v1alpha1"
	"github.com/StepOnce7/tiflow-operator/controllers"
	"github.com/StepOnce7/tiflow-operator/controllers/resources"
	"github.com/StepOnce7/tiflow-operator/controllers/utils"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	_ "sigs.k8s.io/controller-runtime/pkg/controller"
)

// ExecutorMemberManager implements interface of Manager.
type ExecutorMemberManager struct {
	ConfigMapLister   corelisterv1.ConfigMapLister
	ServiceLister     corelisterv1.ServiceLister
	StatefulSetLister appslisters.StatefulSetLister
	executorScale     Scaler
	executorUpgrade   Upgrader
	executorFailover  Failover
}

func NewExecutorMemberManager() controllers.Manager {

	// todo: need a new think about how to access the main logic, and what is needed
	return nil
}

func (m *ExecutorMemberManager) Sync(cluster *v1alpha1.TiflowCluster) error {
	if cluster.Spec.Executor == nil {
		return nil
	}

	// Sync Tiflow Cluster Executor Headless Service
	if err := m.syncExecutorHeadlessService(cluster); err != nil {
		return err
	}

	// Sync Tiflow Cluster Executor StatefulSet
	return m.syncStatefulSet(cluster)
}

func (m *ExecutorMemberManager) syncExecutorConfigMap(cluster *v1alpha1.TiflowCluster, sts *apps.StatefulSet) (*corev1.ConfigMap, error) {
	if cluster.Spec.Executor.Config == nil {
		return nil, nil
	}

	// todo: Need to finish the getExecutorConfigMap logic
	newCfgMap, err := m.getExecutorConfigMap(cluster)
	if err != nil {
		return nil, err
	}

	var inUseName string
	// todo: Need to finish the FindConfigMapVolume logic
	klog.V(3).Info("get executor in use config mao name: ", inUseName)

	// todo: Need to finish the UpdateConfigMapIfNeed Logic
	err = resources.UpdateConfigMapIfNeed(m.ConfigMapLister, v1alpha1.ConfigUpdateStrategyInPlace, inUseName, newCfgMap)
	if err != nil {
		return nil, err
	}

	// todo: Need to finish Control.CreateOrUpdateConfigMap
	return nil, nil
}

func (m *ExecutorMemberManager) syncExecutorHeadlessService(cluster *v1alpha1.TiflowCluster) error {

	clusterNameSpace := cluster.GetNamespace()
	clusterName := cluster.GetName()

	newSvc := m.getNewExecutorHeadlessService(cluster)
	// todo: something may be modify about serviceLister
	oldSvcTmp, err := m.ServiceLister().Services(clusterNameSpace).Get(clusterName)
	if errors.IsNotFound(err) {
		err = utils.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		// todo: create new one
		return nil
	}

	if err != nil {
		return fmt.Errorf("syncExecutorHeadlessService: failed to get svc %s for cluster %s/%s, error: %s", "executor service", clusterNameSpace, clusterName, err)
	}

	oldSvc := oldSvcTmp.DeepCopy()
	equal, err := utils.ServiceEqual(newSvc, oldSvc)
	if !equal {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		err = utils.SetServiceLastAppliedConfigAnnotation(&svc)
		if err != nil {
			return err
		}
		// todo: update service
		return nil
	}

	return nil
}

func (m *ExecutorMemberManager) syncStatefulSet(cluster *v1alpha1.TiflowCluster) error {

	clusterNamespace := cluster.GetNamespace()
	clusterName := cluster.GetName()

	// todo: something may be modify about StatefulSetLister
	oldStsTmp, err := m.StatefulSetLister.StatefulSets(clusterNamespace).Get(clusterName)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncStatefulSet: failed to get sts %s for cluster %s/%s, error: %s", "executor statefulSet", clusterNamespace, clusterName, err)
	}

	stsNotExist := errors.IsNotFound(err)
	oldSts := oldStsTmp.DeepCopy()

	// failed to sync executor status will not affect subsequent logic, just print the errors.
	if err := m.syncExecutorStatus(cluster, oldSts); err != nil {
		klog.Errorf("failed to sync TiflowCluster : [%s/%s]'s executor status, error: %v",
			clusterNamespace, clusterName, err)
	}

	// todo: Paused if need

	cfgMap, err := m.syncExecutorConfigMap(cluster, oldSts)
	if err != nil {
		return err
	}

	newSts, err := m.getNewExecutorStatefulSet(cluster, cfgMap)
	if err != nil {
		return err
	}

	if stsNotExist {
		// todo: set annotation
		// todo: create new statefulSet
		return nil
	}

	// todo: update new statefulSet
	return resources.UpdateStatefulSetWithPreCheck(cluster, "todo", newSts, oldSts)
}

func (m *ExecutorMemberManager) getExecutorConfigMap(cluster *v1alpha1.TiflowCluster) (*corev1.ConfigMap, error) {
	if cluster.Spec.Executor.Config == nil {
		return nil, nil
	}

	config := cluster.Spec.Executor.Config.DeepCopy()
	configText, err := config.MarshalTOML()
	if err != nil {
		return nil, err
	}

	data := make(map[string]string)
	data["config-file"] = string(configText)

	name := fmt.Sprintf("%s-tiflow-executor", cluster.Name)

	// todo: How to get InstanceName、Labels and OwnerReferences
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       cluster.Namespace,
			OwnerReferences: []metav1.OwnerReference{},
		},
		Data: data,
	}

	return cm, nil
}

func (m *ExecutorMemberManager) getNewExecutorHeadlessService(cluster *v1alpha1.TiflowCluster) *corev1.Service {
	return nil
}

func (m *ExecutorMemberManager) getNewExecutorStatefulSet(cluster *v1alpha1.TiflowCluster, cfgMap *corev1.ConfigMap) (*apps.StatefulSet, error) {
	return nil, nil
}

func (m *ExecutorMemberManager) syncExecutorStatus(cluster *v1alpha1.TiflowCluster, sts *apps.StatefulSet) error {

	// skip if not created yet
	if sts == nil {
		return nil
	}

	// nn old statefulSet exists
	//clusterNameSpace := cluster.GetNamespace()
	//clusterName := cluster.GetName()

	// update the status of statefulSet which created by executor in the cluster
	cluster.Status.Executor.StatefulSet = &sts.Status

	// todo: How to get Synced info
	// todo: Need to check if the current sts are updating
	cluster.Status.Executor.Phase = v1alpha1.NormalPhase

	// todo: Get information about the Executor Members, FailureMembers and FailoverUID through the Master API
	// todo: Or may be get info through the Sts Status
	// TOBE

	// todo: Need to get the info of the running version image in the cluster
	cluster.Status.Executor.Image = ""

	// todo: Need to get the info of volumes which running container has bound
	cluster.Status.Executor.Volumes = nil
	return nil
}
