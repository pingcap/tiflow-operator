package tiflowcluster

import (
	"context"

	"github.com/StepOnce7/tiflow-operator/api/v1alpha1"
	"github.com/StepOnce7/tiflow-operator/pkg/manager"
	"github.com/StepOnce7/tiflow-operator/pkg/manager/member"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ControlInterface implements the control logic for updating TiflowClusters and their children StatefulSets.
// It is implemented as an interface to allow for extensions that provide different semantics.
// Currently, there is only one implementation.
type ControlInterface interface {
	// UpdateTiflowCluster implements the control logic for StatefulSet creation, update, and deletion
	UpdateTiflowCluster(ctx context.Context, cluster *v1alpha1.TiflowCluster) error
}

// NewDefaultTiflowClusterControl returns a new instance of the default implementation DMClusterControlInterface that
// implements the documented semantics for DMClusters.
func NewDefaultTiflowClusterControl(cli client.Client) ControlInterface {
	return &defaultTiflowClusterControl{
		cli,
		member.NewMasterMemberManager(cli),
	}
}

type defaultTiflowClusterControl struct {
	cli                 client.Client
	masterMemberManager manager.TiflowManager
}

// UpdateTiflowCluster executes the core logic loop for a tiflowcluster.
func (c *defaultTiflowClusterControl) UpdateTiflowCluster(ctx context.Context, t *v1alpha1.TiflowCluster) error {
	//c.defaulting(t)
	//if !c.validate(t) {
	//	return nil // fatal error, no need to retry on invalid object
	//}

	var errs []error
	oldStatus := t.Status.DeepCopy()

	if err := c.updateTiflowCluster(ctx, t); err != nil {
		errs = append(errs, err)
	}

	if apiequality.Semantic.DeepEqual(&t.Status, oldStatus) {
		return errorutils.NewAggregate(errs)
	}
	// TODO: add status updater

	return errorutils.NewAggregate(errs)
}

func (c *defaultTiflowClusterControl) updateTiflowCluster(ctx context.Context, tc *v1alpha1.TiflowCluster) error {
	var errs []error

	// works that should be done to make the dm-master cluster current state match the desired state:
	//   - create or update the dm-master service
	//   - create or update the dm-master headless service
	//   - create the dm-master statefulset
	//   - sync dm-master cluster status from dm-master to DMCluster object
	//   - set two annotations to the first dm-master member:
	// 	   - label.Bootstrapping
	// 	   - label.Replicas
	//   - upgrade the dm-master cluster
	//   - scale out/in the dm-master cluster
	//   - failover the dm-master cluster
	if err := c.masterMemberManager.Sync(ctx, tc); err != nil {
		errs = append(errs, err)
	}

	return errorutils.NewAggregate(errs)
}
