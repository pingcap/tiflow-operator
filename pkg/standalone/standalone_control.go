package standalone

import (
	"context"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ControlInterface implements the control logic for updating Standalone and their children StatefulSets.
// It is implemented as an interface to allow for extensions that provide different semantics.
// Currently, there is only one implementation.
type ControlInterface interface {
	UpdateStandalone(ctx context.Context, cluster *v1alpha1.Standalone) (ctrl.Result, error)
}

type defaultStandaloneControl struct {
	cli          client.Client
	frameManager StandaloneManager
	userManager  StandaloneManager
}

func NewDefaultStandaloneControl(cli client.Client, schema *runtime.Scheme) ControlInterface {
	return &defaultStandaloneControl{
		cli,
		NewFrameManager(cli, schema),
		NewUserManager(cli, schema),
	}
}

// UpdateStandalone executes the core logic loop for a Standalone.
func (c *defaultStandaloneControl) UpdateStandalone(ctx context.Context, instance *v1alpha1.Standalone) (ctrl.Result, error) {

	if result, err := c.frameManager.Sync(ctx, instance); err != nil {
		return result, err
	}

	if result, err := c.userManager.Sync(ctx, instance); err != nil {
		return result, err
	}

	return ctrl.Result{}, nil
}
