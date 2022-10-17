package status

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

func GetSyncStatus(syncName v1alpha1.SyncTypeName, tc *v1alpha1.TiflowClusterStatus, member v1alpha1.MemberType) v1alpha1.SyncTypeStatus {
	if member == v1alpha1.TiFlowMasterMemberType {
		return findMasterSyncTypeStatus(syncName, &tc.Master)
	}
	return findExecutorSyncTypeStatus(syncName, &tc.Executor)
}

func Ongoing(syncName v1alpha1.SyncTypeName, tc *v1alpha1.TiflowClusterStatus, member v1alpha1.MemberType, message string) {
	if member == v1alpha1.TiFlowMasterMemberType {
		setMasterSyncTypeStatus(syncName, v1alpha1.Ongoing, &tc.Master, message, metav1.Now())
		return
	}
	setExecutorSyncTypeStatus(syncName, v1alpha1.Ongoing, &tc.Executor, message, metav1.Now())
}

func Completed(syncName v1alpha1.SyncTypeName, tc *v1alpha1.TiflowClusterStatus, member v1alpha1.MemberType, message string) {
	if member == v1alpha1.TiFlowMasterMemberType {
		setMasterSyncTypeStatus(syncName, v1alpha1.Completed, &tc.Master, message, metav1.Now())
		return
	}
	setExecutorSyncTypeStatus(syncName, v1alpha1.Completed, &tc.Executor, message, metav1.Now())
}

func Unknown(syncName v1alpha1.SyncTypeName, tc *v1alpha1.TiflowClusterStatus, member v1alpha1.MemberType, message string) {
	if member == v1alpha1.TiFlowMasterMemberType {
		setMasterSyncTypeStatus(syncName, v1alpha1.Unknown, &tc.Master, message, metav1.Now())
		return
	}
	setExecutorSyncTypeStatus(syncName, v1alpha1.Unknown, &tc.Executor, message, metav1.Now())
}

func Failed(syncName v1alpha1.SyncTypeName, tc *v1alpha1.TiflowClusterStatus, member v1alpha1.MemberType, message string) {
	if member == v1alpha1.TiFlowMasterMemberType {
		setMasterSyncTypeStatus(syncName, v1alpha1.Failed, &tc.Master, message, metav1.Now())
		return
	}
	setExecutorSyncTypeStatus(syncName, v1alpha1.Failed, &tc.Executor, message, metav1.Now())
}

func setMasterSyncTypeStatus(syncName v1alpha1.SyncTypeName, syncStatus v1alpha1.SyncTypeStatus, master *v1alpha1.MasterStatus, message string, now metav1.Time) {
	sync := findOrCreateMasterSyncType(syncName, master, message)
	sync.Status = syncStatus
	sync.LastUpdateTime = now
}

func setExecutorSyncTypeStatus(syncName v1alpha1.SyncTypeName, syncStatus v1alpha1.SyncTypeStatus, executor *v1alpha1.ExecutorStatus, message string, now metav1.Time) {
	sync := findOrCreateExecutorSyncType(syncName, executor, message)
	sync.Status = syncStatus
	sync.LastUpdateTime = now
}

func findOrCreateMasterSyncType(syncName v1alpha1.SyncTypeName, master *v1alpha1.MasterStatus, message string) *v1alpha1.ClusterSyncType {
	pos := findPos(syncName, master.SyncTypes)
	if pos >= 0 {
		master.SyncTypes[pos].Message = message
		return &master.SyncTypes[pos]
	}

	master.SyncTypes = append(master.SyncTypes, v1alpha1.ClusterSyncType{
		Name:           syncName,
		Message:        message,
		Status:         v1alpha1.Unknown,
		LastUpdateTime: metav1.Now(),
	})

	return &master.SyncTypes[len(master.SyncTypes)-1]
}

func findOrCreateExecutorSyncType(syncName v1alpha1.SyncTypeName, executor *v1alpha1.ExecutorStatus, message string) *v1alpha1.ClusterSyncType {
	pos := findPos(syncName, executor.SyncTypes)
	if pos >= 0 {
		executor.SyncTypes[pos].Message = message
		return &executor.SyncTypes[pos]
	}

	executor.SyncTypes = append(executor.SyncTypes, v1alpha1.ClusterSyncType{
		Name:           syncName,
		Message:        message,
		Status:         v1alpha1.Unknown,
		LastUpdateTime: metav1.Now(),
	})

	return &executor.SyncTypes[len(executor.SyncTypes)-1]
}

func findMasterSyncTypeStatus(syncName v1alpha1.SyncTypeName, master *v1alpha1.MasterStatus) v1alpha1.SyncTypeStatus {
	pos := findPos(syncName, master.SyncTypes)
	if pos >= 0 {
		return master.SyncTypes[pos].Status
	}
	return v1alpha1.Unknown
}

func findExecutorSyncTypeStatus(syncName v1alpha1.SyncTypeName, executor *v1alpha1.ExecutorStatus) v1alpha1.SyncTypeStatus {
	pos := findPos(syncName, executor.SyncTypes)
	if pos >= 0 {
		return executor.SyncTypes[pos].Status
	}
	return v1alpha1.Unknown
}
