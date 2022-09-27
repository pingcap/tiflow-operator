package condition

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

func InitConditionsIfNeed(status *v1alpha1.TiflowClusterStatus, now metav1.Time) {
	if status.ClusterConditions == nil {
		status.ClusterConditions = []v1alpha1.TiflowClusterCondition{}
	}
	return
}

func True(ctype v1alpha1.TiflowClusterConditionType, conds []v1alpha1.TiflowClusterCondition) bool {
	index := pos(ctype, conds)
	if index == -1 || conds[index].Status == metav1.ConditionUnknown {
		return false
	}

	return conds[index].Status == metav1.ConditionTrue
}

func False(ctype v1alpha1.TiflowClusterConditionType, conds []v1alpha1.TiflowClusterCondition) bool {
	index := pos(ctype, conds)
	if index == -1 || conds[index].Status == metav1.ConditionUnknown {
		return false
	}

	return conds[index].Status == metav1.ConditionFalse
}

func Unknown(ctype v1alpha1.TiflowClusterConditionType, conds []v1alpha1.TiflowClusterCondition) bool {
	index := pos(ctype, conds)
	if index == -1 {
		return false
	}

	return conds[index].Status == metav1.ConditionUnknown
}

func SetFalse(ctype v1alpha1.TiflowClusterConditionType, status *v1alpha1.TiflowClusterStatus, now metav1.Time) {
	setConditionStatus(ctype, metav1.ConditionFalse, status, now)
}

func SetTrue(ctype v1alpha1.TiflowClusterConditionType, status *v1alpha1.TiflowClusterStatus, now metav1.Time) {
	setConditionStatus(ctype, metav1.ConditionTrue, status, now)
}

func setConditionStatus(ctype v1alpha1.TiflowClusterConditionType, status metav1.ConditionStatus, clusterStatus *v1alpha1.TiflowClusterStatus, now metav1.Time) {
	cond := findOrCreate(ctype, clusterStatus)

	if cond.Status == status {
		return
	}

	cond.Status = status
	cond.LastTransitionTime = now
}

func findOrCreate(ctype v1alpha1.TiflowClusterConditionType, clusterStatus *v1alpha1.TiflowClusterStatus) *v1alpha1.TiflowClusterCondition {
	pos := pos(ctype, clusterStatus.ClusterConditions)
	if pos >= 0 {
		return &clusterStatus.ClusterConditions[pos]
	}

	clusterStatus.ClusterConditions = append(clusterStatus.ClusterConditions, v1alpha1.TiflowClusterCondition{
		Type:               ctype,
		Status:             metav1.ConditionUnknown,
		LastTransitionTime: metav1.Now(),
	})

	return &clusterStatus.ClusterConditions[len(clusterStatus.ClusterConditions)-1]
}

func pos(ctype v1alpha1.TiflowClusterConditionType, conds []v1alpha1.TiflowClusterCondition) int {
	for i := range conds {
		if conds[i].Type == ctype {
			return i
		}
	}

	return -1
}
