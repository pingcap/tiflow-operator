package status

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

// SetTiflowClusterStatusOnFirstReconcile will set phase of TiflowCluster as Starting on reconcile first
func SetTiflowClusterStatusOnFirstReconcile(status *v1alpha1.TiflowClusterStatus) {
	if status.ClusterPhase != "" {
		return
	}
	SetMasterClusterStatusOnFirstReconcile(&status.Master)
	SetExecutorClusterStatusOnFirstReconcile(&status.Executor)
	status.ClusterPhase = v1alpha1.ClusterStarting
	status.LastTransitionTime = metav1.Now()
}

func SetTiflowClusterStatus(status *v1alpha1.TiflowClusterStatus) {
	SetMasterClusterStatus(&status.Master)
	SetExecutorClusterStatus(&status.Executor)

	masterPhase, executorPhase := status.Master.Phase, status.Executor.Phase
	switch {
	case masterPhase == "" || executorPhase == "":
		status.ClusterPhase = v1alpha1.ClusterPending
	case masterPhase == v1alpha1.MasterFailed || executorPhase == v1alpha1.ExecutorFailed:
		status.ClusterPhase = v1alpha1.ClusterFailed
	case masterPhase == v1alpha1.MasterUnknown || executorPhase == v1alpha1.ExecutorUnknown:
		status.ClusterPhase = v1alpha1.ClusterUnknown
	case masterPhase == v1alpha1.MasterRunning && executorPhase == v1alpha1.ExecutorRunning:
		status.ClusterPhase = v1alpha1.ClusterCompleted
	default:
		status.ClusterPhase = v1alpha1.ClusterReconciling
	}

	status.LastTransitionTime = metav1.Now()
	return
}
