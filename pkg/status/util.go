package status

import (
	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

func findPos(syncName v1alpha1.SyncTypeName, syncTypes []v1alpha1.ClusterSyncType) int {
	for i := range syncTypes {
		if syncTypes[i].Name == syncName {
			return i
		}
	}

	return -1
}
