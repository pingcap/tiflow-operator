package status

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

type MasterSyncManager struct {
	Status *v1alpha1.MasterStatus
}

func NewMasterSyncTypeManager(status *v1alpha1.MasterStatus) SyncTypeManager {
	return &MasterSyncManager{
		Status: status,
	}
}

func (mm *MasterSyncManager) Ongoing(syncName v1alpha1.SyncTypeName, message string) {
	mm.setMasterSyncTypeStatus(syncName, v1alpha1.Ongoing, message, metav1.Now())
}

func (mm *MasterSyncManager) Complied(syncName v1alpha1.SyncTypeName, message string) {
	mm.setMasterSyncTypeStatus(syncName, v1alpha1.Completed, message, metav1.Now())
}

func (mm *MasterSyncManager) Failed(syncName v1alpha1.SyncTypeName, message string) {
	mm.setMasterSyncTypeStatus(syncName, v1alpha1.Failed, message, metav1.Now())
}

func (mm *MasterSyncManager) Unknown(syncName v1alpha1.SyncTypeName, message string) {
	mm.setMasterSyncTypeStatus(syncName, v1alpha1.Unknown, message, metav1.Now())
}

func setMasterClusterStatusOnFirstReconcile(masterStatus *v1alpha1.MasterStatus) {
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

// updateMasterClusterStatus
// todo: need to check for all OperatorActions or for just the 0th element
// This depends on our logic for updating Status
func updateMasterClusterStatus(masterStatus *v1alpha1.MasterStatus) {
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

func (mm *MasterSyncManager) setMasterSyncTypeStatus(syncName v1alpha1.SyncTypeName, syncStatus v1alpha1.SyncTypeStatus, message string, now metav1.Time) {
	sync := mm.findOrCreateMasterSyncType(syncName, message)
	sync.Status = syncStatus
	sync.LastUpdateTime = now
}

func (mm *MasterSyncManager) findOrCreateMasterSyncType(syncName v1alpha1.SyncTypeName, message string) *v1alpha1.ClusterSyncType {
	pos := findPos(syncName, mm.Status.SyncTypes)
	if pos >= 0 {
		mm.Status.SyncTypes[pos].Message = message
		return &mm.Status.SyncTypes[pos]
	}

	mm.Status.SyncTypes = append(mm.Status.SyncTypes, v1alpha1.ClusterSyncType{
		Name:           syncName,
		Message:        message,
		Status:         v1alpha1.Unknown,
		LastUpdateTime: metav1.Now(),
	})

	return &mm.Status.SyncTypes[len(mm.Status.SyncTypes)-1]
}
