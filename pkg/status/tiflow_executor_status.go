package status

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

type ExecutorSyncTypeManger struct {
	Status *v1alpha1.ExecutorStatus
}

func NewExecutorSyncTypeManager(status *v1alpha1.ExecutorStatus) SyncTypeManager {
	return &ExecutorSyncTypeManger{
		Status: status,
	}
}

func (em *ExecutorSyncTypeManger) Ongoing(syncName v1alpha1.SyncTypeName, message string) {
	em.setExecutorSyncTypeStatus(syncName, v1alpha1.Ongoing, message, metav1.Now())
}

func (em *ExecutorSyncTypeManger) Complied(syncName v1alpha1.SyncTypeName, message string) {
	em.setExecutorSyncTypeStatus(syncName, v1alpha1.Completed, message, metav1.Now())
}

func (em *ExecutorSyncTypeManger) Unknown(syncName v1alpha1.SyncTypeName, message string) {
	em.setExecutorSyncTypeStatus(syncName, v1alpha1.Unknown, message, metav1.Now())
}

func (em *ExecutorSyncTypeManger) Failed(syncName v1alpha1.SyncTypeName, message string) {
	em.setExecutorSyncTypeStatus(syncName, v1alpha1.Failed, message, metav1.Now())
}

func setExecutorClusterStatusOnFirstReconcile(executorStatus *v1alpha1.ExecutorStatus) {
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

// updateExecutorClusterStatus
// todo: need to check for all OperatorActions or for just the 0th element
// This depends on our logic for updating Status
func updateExecutorClusterStatus(executorStatus *v1alpha1.ExecutorStatus) {
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

func (em *ExecutorSyncTypeManger) setExecutorSyncTypeStatus(syncName v1alpha1.SyncTypeName, syncStatus v1alpha1.SyncTypeStatus, message string, now metav1.Time) {
	sync := em.findOrCreateExecutorSyncType(syncName, message)
	sync.Status = syncStatus
	sync.LastUpdateTime = now
}

func (em *ExecutorSyncTypeManger) findOrCreateExecutorSyncType(syncName v1alpha1.SyncTypeName, message string) *v1alpha1.ClusterSyncType {
	pos := findPos(syncName, em.Status.SyncTypes)
	if pos >= 0 {
		em.Status.SyncTypes[pos].Message = message
		return &em.Status.SyncTypes[pos]
	}

	em.Status.SyncTypes = append(em.Status.SyncTypes, v1alpha1.ClusterSyncType{
		Name:           syncName,
		Message:        message,
		Status:         v1alpha1.Unknown,
		LastUpdateTime: metav1.Now(),
	})

	return &em.Status.SyncTypes[len(em.Status.SyncTypes)-1]
}
