package tiflowcluster

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

const (
	// Reasons for TiflowCluster conditions.

	// Ready
	Ready = "Ready"
	// StatefulSetNotUpToDate is added when one of statefulsets is not up to date.
	StatfulSetNotUpToDate = "StatefulSetNotUpToDate"
	// MasterNotUpYet is added when one of tiflow-master members is not up.
	MasterNotUpYet = "TiflowMasterNotUp"
	// ExecutorNotUpYet is added when one of tiflow-executor members is not up.
	ExecutorNotUpYet = "TiflowExecutorNotUp"
)

// NewTiflowClusterCondition creates a new tiflowcluster condition.
func NewTiflowClusterCondition(condType string, status v1.ConditionStatus, reason, message string) *v1alpha1.TiflowClusterCondition {
	return &v1alpha1.TiflowClusterCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// GetTiflowClusterCondition returns the condition with the provided type.
func GetTiflowClusterCondition(status v1alpha1.TiflowClusterStatus, condType string) *v1alpha1.TiflowClusterCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// SetTiflowClusterCondition updates the tiflow cluster to include the provided condition. If the condition that
// we are about to add already exists and has the same status and reason then we are not going to update.
func SetTiflowClusterCondition(status *v1alpha1.TiflowClusterStatus, condition v1alpha1.TiflowClusterCondition) {
	currentCond := GetTiflowClusterCondition(*status, condition.Type)
	if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason {
		return
	}
	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}
	newConditions := filterOutCondition(status.Conditions, condition.Type)
	status.Conditions = append(newConditions, condition)
}

// filterOutCondition returns a new slice of tiflowcluster conditions without conditions with the provided type.
func filterOutCondition(conditions []v1alpha1.TiflowClusterCondition, condType string) []v1alpha1.TiflowClusterCondition {
	var newConditions []v1alpha1.TiflowClusterCondition
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}

// GetTiflowClusterReadyCondition extracts the tiflowcluster ready condition from the given status and returns that.
// Returns nil if the condition is not present.
func GetTiflowClusterReadyCondition(status v1alpha1.TiflowClusterStatus) *v1alpha1.TiflowClusterCondition {
	return GetTiflowClusterCondition(status, Ready)
}
