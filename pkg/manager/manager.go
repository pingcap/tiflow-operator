package manager

import (
	"context"

	pingcapcomv1alpha1 "github.com/StepOnce7/tiflow-operator/api/v1alpha1"
)

type TiflowManager interface {
	// Sync implements the logic for syncing tiflowcluster.
	Sync(context.Context, *pingcapcomv1alpha1.TiflowCluster) error
}
