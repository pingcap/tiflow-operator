package prune

import (
	"context"
	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

type PVCPruner interface {
	Prune(ctx context.Context, tc *v1alpha1.TiflowCluster) error
}
