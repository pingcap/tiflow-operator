package status

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

func SetExecutorClusterStatusOnFirstReconcile(executorStatus *v1alpha1.ExecutorStatus) {
	InitExecutorClusterSyncTypesIfNeed(executorStatus)
	if executorStatus.Phase != "" {
		return
	}

	executorStatus.Phase = v1alpha1.ExecutorStarting
	executorStatus.Message = "Starting..., tiflow-executor on first reconcile. Just a moment"
	executorStatus.LastTransitionTime = metav1.Now()
	return
}

func InitExecutorClusterSyncTypesIfNeed(executorStatus *v1alpha1.ExecutorStatus) {
	if executorStatus.SyncTypes == nil {
		executorStatus.SyncTypes = []v1alpha1.ClusterSyncType{}
	}
	return
}

// SetExecutorClusterStatus
// todo: need to check for all OperatorActions or for just the 0th element
// This depends on our logic for updating Status
func SetExecutorClusterStatus(executorStatus *v1alpha1.ExecutorStatus) {
	InitExecutorClusterSyncTypesIfNeed(executorStatus)
	for _, sync := range executorStatus.SyncTypes {
		switch sync.Status {
		case v1alpha1.Failed:
			executorStatus.Phase = v1alpha1.ExecutorFailed
		case v1alpha1.Unknown:
			executorStatus.Phase = v1alpha1.ExecutorUnknown
		case v1alpha1.Completed:
			executorStatus.Phase = v1alpha1.ExecutorRunning
		default:
			executorStatus.Phase = sync.Name.GetExecutorClusterPhase()
		}

		executorStatus.Message = sync.Message
		executorStatus.LastTransitionTime = metav1.Now()
		return
	}

	executorStatus.Phase = v1alpha1.ExecutorRunning
	executorStatus.Message = "Ready..., tiflow-executor reconcile completed successfully. Enjoying..."
	executorStatus.LastTransitionTime = metav1.Now()
	return
}

func setExecutorSyncTypeStatus(syncName v1alpha1.SyncTypeName, syncStatus v1alpha1.SyncTypeStatus, message string, now metav1.Time, status *v1alpha1.ExecutorStatus) {
	sync := findOrCreateExecutorSyncType(syncName, status, message)
	sync.Status = syncStatus
	sync.LastUpdateTime = now
}

func findOrCreateExecutorSyncType(syncName v1alpha1.SyncTypeName, status *v1alpha1.ExecutorStatus, message string) *v1alpha1.ClusterSyncType {
	pos := findPos(syncName, status.SyncTypes)
	if pos >= 0 {
		status.SyncTypes[pos].Message = message
		return &status.SyncTypes[pos]
	}

	status.SyncTypes = append(status.SyncTypes, v1alpha1.ClusterSyncType{
		Name:           syncName,
		Message:        message,
		Status:         v1alpha1.Unknown,
		LastUpdateTime: metav1.Now(),
	})

	return &status.SyncTypes[len(status.SyncTypes)-1]
}
