package status

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

func SetClusterSyncTypeFailed(syncName v1alpha1.SyncTypeName, status v1alpha1.TiflowClusterStatus, member v1alpha1.MemberType) {
	if member == v1alpha1.TiFlowMasterMemberType {
		setMasterSyncTypeStatus(syncName, v1alpha1.Failed, "", metav1.Now(), &status.Master)
	}
	setExecutorSyncTypeStatus(syncName, v1alpha1.Failed, "", metav1.Now(), &status.Executor)
}

func SetClusterSyncTypeComplied(syncName v1alpha1.SyncTypeName, status v1alpha1.TiflowClusterStatus, member v1alpha1.MemberType) {
	if member == v1alpha1.TiFlowMasterMemberType {
		setMasterSyncTypeStatus(syncName, v1alpha1.Completed, "", metav1.Now(), &status.Master)
	}
	setExecutorSyncTypeStatus(syncName, v1alpha1.Completed, "", metav1.Now(), &status.Executor)
}

func SetClusterSyncTypeUnknown(syncName v1alpha1.SyncTypeName, status v1alpha1.TiflowClusterStatus, member v1alpha1.MemberType) {
	if member == v1alpha1.TiFlowMasterMemberType {
		setMasterSyncTypeStatus(syncName, v1alpha1.Unknown, "", metav1.Now(), &status.Master)
	}
	setExecutorSyncTypeStatus(syncName, v1alpha1.Unknown, "", metav1.Now(), &status.Executor)
}

func SetClusterSyncTypeOngoing(syncName v1alpha1.SyncTypeName, status v1alpha1.TiflowClusterStatus, member v1alpha1.MemberType) {
	if member == v1alpha1.TiFlowMasterMemberType {
		setMasterSyncTypeStatus(syncName, v1alpha1.Ongoing, "", metav1.Now(), &status.Master)
	}
	setExecutorSyncTypeStatus(syncName, v1alpha1.Ongoing, "", metav1.Now(), &status.Executor)
}
