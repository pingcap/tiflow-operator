package controllers

import (
	"context"
	"github.com/StepOnce7/tiflow-operator/api/v1alpha1"
)

// Manager implements the logic for syncing tiflow cluster
type Manager interface {
	// Sync    implements the logic for syncing tiflow cluster.
	Sync(ctx context.Context, cluster *v1alpha1.TiflowCluster) error
}
