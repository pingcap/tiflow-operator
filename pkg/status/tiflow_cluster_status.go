package status

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

type TiflowClusterStatusManager struct {
	MasterStatus   SyncTypeManager
	ExecutorStatus SyncTypeManager
}

func NewTiflowClusterStatusManager(clusterStatus *v1alpha1.TiflowClusterStatus) *TiflowClusterStatusManager {
	return &TiflowClusterStatusManager{
		MasterStatus:   NewMasterSyncTypeManager(&clusterStatus.Master),
		ExecutorStatus: NewExecutorSyncTypeManager(&clusterStatus.Executor),
	}
}

// SetTiflowClusterStatusOnFirstReconcile will set phase of TiflowCluster as Starting on reconcile first
func SetTiflowClusterStatusOnFirstReconcile(status *v1alpha1.TiflowClusterStatus) {
	if status.ClusterPhase != "" {
		return
	}
	setMasterClusterStatusOnFirstReconcile(&status.Master)
	setExecutorClusterStatusOnFirstReconcile(&status.Executor)
	status.ClusterPhase = v1alpha1.ClusterStarting
	status.Message = "Starting... tiflow-cluster on first reconcile. Just a moment"
	status.LastTransitionTime = metav1.Now()
}

func (tcsm *TiflowClusterStatusManager) UpdateTiflowClusterStatus(clientSet kubernetes.Interface, tc *v1alpha1.TiflowCluster) {
	tcsm.MasterStatus.UpdateClusterStatus(clientSet, tc)
	tcsm.ExecutorStatus.UpdateClusterStatus(clientSet, tc)

	// masterPhase, executorPhase := status.Master.Phase, status.Executor.Phase
	// switch {
	// case masterPhase == "" || executorPhase == "":
	// 	status.ClusterPhase = v1alpha1.ClusterPending
	// 	status.Message = "errors, no phase information getting from  Master or Executor"
	// case masterPhase == v1alpha1.MasterFailed || executorPhase == v1alpha1.ExecutorFailed:
	// 	status.ClusterPhase = v1alpha1.ClusterFailed
	// 	status.Message = "errors, failed phase for Master or Executor"
	// case masterPhase == v1alpha1.MasterUnknown || executorPhase == v1alpha1.ExecutorUnknown:
	// 	status.ClusterPhase = v1alpha1.ClusterUnknown
	// 	status.Message = "errors, unknown phase for Master or Executor"
	// case masterPhase == v1alpha1.MasterRunning && executorPhase == v1alpha1.ExecutorRunning:
	// 	status.ClusterPhase = v1alpha1.ClusterCompleted
	// 	status.Message = "reconcile completed successfully. Enjoying..."
	// default:
	// 	status.ClusterPhase = v1alpha1.ClusterReconciling
	// 	status.Message = "reconciling... Just a moment"
	// }
	//
	// status.LastTransitionTime = metav1.Now()
	return
}
