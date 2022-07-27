package resources

import (
	"github.com/StepOnce7/tiflow-operator/api/v1alpha1"
	corev1 "k8s.io/api/apps/v1"
)

// UpdateStatefulSetWithPreCheck todo: impl this logic
func UpdateStatefulSetWithPreCheck(cluster *v1alpha1.TiflowCluster, resaon string, newSts, oldSts *corev1.StatefulSet) error {
	return nil
}
