package member

import (
	"context"

	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	skipReasonScalerPVCNotFound             = "scaler: pvc is not found"
	skipReasonScalerAnnIsNil                = "scaler: pvc annotations is nil"
	skipReasonScalerAnnDeferDeletingIsEmpty = "scaler: pvc annotations defer deleting is empty"
)

// Scaler implements the logic for scaling out or scaling in the cluster.
type Scaler interface {
	// Scale scales the cluster. It does nothing if scaling is not needed.
	Scale(ctx context.Context, meta metav1.Object, actual *apps.StatefulSet, desired *apps.StatefulSet) error
	// ScaleOut scales out the cluster
	ScaleOut(ctx context.Context, meta metav1.Object, actual *apps.StatefulSet, desired *apps.StatefulSet) error
	// ScaleIn scales in the cluster
	ScaleIn(ctx context.Context, meta metav1.Object, actual *apps.StatefulSet, desired *apps.StatefulSet) error
	// SetReplicas sets the desired replicas for statefulSet without waiting for
	// new pods to be created or to become healthy. And update it.
	SetReplicas(ctx context.Context, actual *apps.StatefulSet, desired uint) error
	// WaitUntilRunning blocks until the target statefulSet has the expected number of pods running but not necessarily ready
	WaitUntilRunning(ctx context.Context) error
	// WaitUntilHealthy blocks until the target stateful set has exactly `scale` healthy replicas.
	WaitUntilHealthy(ctx context.Context, scale uint) error
}

type PVCPruner interface {
	Prune(ctx context.Context) error
	IsPrune() bool
}

type PVCMounter interface {
	Mounter(ctx context.Context) error
	IsMount() bool
}
