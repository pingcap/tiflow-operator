package controllers

import (
	"fmt"
	"github.com/StepOnce7/tiflow-operator/api/v1alpha1"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

func NewExecutorMemberManager() Manager {

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
	return nil, nil
}

func (m *ExecutorMemberManager) syncExecutorHeadlessService(cluster *v1alpha1.TiflowCluster) error {

	clusterNameSpace := cluster.GetNamespace()
	clusterName := cluster.GetName()

	newSvc := m.getNewExecutorHeadlessService(cluster)
	// todo: something may be modify about serviceLister
	oldSvcTmp, err := m.ServiceLister().Services(clusterNameSpace).Get(clusterName)
	if errors.IsNotFound(err) {
		err = SetServiceLastAppliedConfigAnnotation(newSvc)
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
	equal, err := ServiceEqual(newSvc, oldSvc)
	if !equal {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		err = SetServiceLastAppliedConfigAnnotation(&svc)
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

	oldStsTmp, err := m.StatefulSetLister.StatefulSets(clusterNamespace).Get(clusterName)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncStatefulSet: failed to get sts %s for cluster %s/%s, error: %s", "executor statefulSet", clusterNamespace, clusterName, err)
	}

	stsNotExist := errors.IsNotFound(err)
	oldSts := oldStsTmp.DeepCopy()

	if err := m.syncExecutorStatus(cluster, oldSts); err != nil {
		klog.Errorf("failed to sync executor stsus: [%s/%s]'s ticdc status, error: %v",
			clusterNamespace, clusterName, err)
	}

	// todo: Paused

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
	return UpdateStatefulSetWithPreCheck(cluster, "todo", newSts, oldSts)
}

func (m *ExecutorMemberManager) getExecutorConfigMap(cluster *v1alpha1.TiflowCluster) (*corev1.ConfigMap, error) {

	return nil, nil
}

func (m *ExecutorMemberManager) getNewExecutorHeadlessService(cluster *v1alpha1.TiflowCluster) *corev1.Service {
	return nil
}

func (m *ExecutorMemberManager) getNewExecutorStatefulSet(cluster *v1alpha1.TiflowCluster, cfgMap *corev1.ConfigMap) (*apps.StatefulSet, error) {
	return nil, nil
}

func (m *ExecutorMemberManager) syncExecutorStatus(cluster *v1alpha1.TiflowCluster, sts *apps.StatefulSet) error {
	return nil
}
