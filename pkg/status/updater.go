package status

import (
	"context"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

type Updater interface {
	Update(context.Context, *v1alpha1.TiflowCluster) error
}
