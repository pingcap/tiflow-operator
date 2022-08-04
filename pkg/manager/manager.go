package manager

import (
	"context"

	pingcapcomv1alpha1 "github.com/pingcap/tiflow-operator/api/v1alpha1"
)

type TiflowManager interface {
	// Sync implements the logic for syncing tiflowCluster.
	Sync(context.Context, *pingcapcomv1alpha1.TiflowCluster) error
}
