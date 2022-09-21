package status

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

func SetMasterClusterStatusOnFirstReconcile(masterStatus *v1alpha1.MasterStatus) {
	InitMasterClusterSyncTypesIfNeed(masterStatus)
	if masterStatus.Phase != "" {
		return
	}

	masterStatus.Phase = v1alpha1.MasterStarting
	masterStatus.Message = "Starting... tiflow-master on first reconcile. Just a moment"
	masterStatus.LastTransitionTime = metav1.Now()
	return
}

func InitMasterClusterSyncTypesIfNeed(masterStatus *v1alpha1.MasterStatus) {
	if masterStatus.SyncTypes == nil {
		masterStatus.SyncTypes = []v1alpha1.ClusterSyncType{}
	}
	return
}

// SetMasterClusterStatus
// todo: need to check for all OperatorActions or for just the 0th element
// This depends on our logic for updating Status
func SetMasterClusterStatus(masterStatus *v1alpha1.MasterStatus) {
	InitMasterClusterSyncTypesIfNeed(masterStatus)
	for _, sync := range masterStatus.SyncTypes {
		switch sync.Status {
		case v1alpha1.Failed:
			masterStatus.Phase = v1alpha1.MasterFailed
		case v1alpha1.Unknown:
			masterStatus.Phase = v1alpha1.MasterUnknown
		case v1alpha1.Completed:
			masterStatus.Phase = v1alpha1.MasterRunning
		default:
			masterStatus.Phase = sync.Name.GetMasterClusterPhase()
		}

		masterStatus.Message = sync.Message
		masterStatus.LastTransitionTime = metav1.Now()
		return
	}

	masterStatus.Phase = v1alpha1.MasterRunning
	masterStatus.Message = "Ready..., tiflow-master reconcile completed successfully. Enjoying..."
	masterStatus.LastTransitionTime = metav1.Now()
	return
}

func setMasterSyncTypeStatus(syncName v1alpha1.SyncTypeName, syncStatus v1alpha1.SyncTypeStatus, message string, now metav1.Time, status *v1alpha1.MasterStatus) {
	sync := findOrCreateMasterSyncType(syncName, status, message)
	sync.Status = syncStatus
	sync.LastUpdateTime = now
}

func findOrCreateMasterSyncType(syncName v1alpha1.SyncTypeName, status *v1alpha1.MasterStatus, message string) *v1alpha1.ClusterSyncType {
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
