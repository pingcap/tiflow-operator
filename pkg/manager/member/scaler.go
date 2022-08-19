package member

import (
	"fmt"

	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

const (
	skipReasonScalerPVCNotFound             = "scaler: pvc is not found"
	skipReasonScalerAnnIsNil                = "scaler: pvc annotations is nil"
	skipReasonScalerAnnDeferDeletingIsEmpty = "scaler: pvc annotations defer deleting is empty"
)

// Scaler implements the logic for scaling out or scaling in the cluster.
type Scaler interface {
	// Scale scales the cluster. It does nothing if scaling is not needed.
	Scale(meta metav1.Object, actual *apps.StatefulSet, desired *apps.StatefulSet) error
	// ScaleOut scales out the cluster
	ScaleOut(meta metav1.Object, actual *apps.StatefulSet, desired *apps.StatefulSet) error
	// ScaleIn scales in the cluster
	ScaleIn(meta metav1.Object, actual *apps.StatefulSet, desired *apps.StatefulSet) error
}

func ordinalPodName(memberType v1alpha1.MemberType, tcName string, ordinal int32) string {
	return fmt.Sprintf("%s-%s-%d", tcName, memberType, ordinal)
}
