package standalone

import (
	"context"
	pingcapcomv1alpha1 "github.com/StepOnce7/tiflow-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type StandaloneManager interface {
	// Sync implements the logic for syncing standalone.
	Sync(context.Context, *pingcapcomv1alpha1.Standalone) (ctrl.Result, error)
}
