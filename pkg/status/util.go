package status

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

func masterMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tiflow-master", clusterName)
}

func executorMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tiflow-executor", clusterName)
}

func findPos(syncName v1alpha1.SyncTypeName, syncTypes []v1alpha1.ClusterSyncType) int {
	for i := range syncTypes {
		if syncTypes[i].Name == syncName {
			return i
		}
	}

	return -1
}

func isCreating() bool {

	return true
}

func isScaling() bool {
	return true
}

// StatefulSetIsUpgrading confirms whether the statefulSet is upgrading phase
func isUpgrading(set *appsv1.StatefulSet) bool {
	if set.Status.CurrentRevision != set.Status.UpdateRevision {
		return true
	}
	if set.Generation > set.Status.ObservedGeneration && *set.Spec.Replicas == set.Status.Replicas {
		return true
	}
	return false
}
