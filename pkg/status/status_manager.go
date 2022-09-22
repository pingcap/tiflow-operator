package status

import "github.com/pingcap/tiflow-operator/api/v1alpha1"

type SyncTypeManager interface {
	SetClusterSyncTypeOngoing(v1alpha1.SyncTypeName, string)
	SetClusterSyncTypeComplied(v1alpha1.SyncTypeName, string)
	SetClusterSyncTypeFailed(v1alpha1.SyncTypeName, string)
	SetClusterSyncTypeUnknown(v1alpha1.SyncTypeName, string)
}
