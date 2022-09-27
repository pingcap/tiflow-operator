package status

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

type realStatusUpdater struct {
	cli client.Client
}

func NewRealStatusUpdater(cli client.Client) Updater {
	return &realStatusUpdater{
		cli,
	}
}

func (c *realStatusUpdater) Update(ctx context.Context, tc *v1alpha1.TiflowCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	status := tc.Status.DeepCopy()

	// don't wait due to limited number of clients, but backoff after the default number of steps
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var updateErr error
		// tc will be updated in Update function, must use cli.Status().Update instead of cli.Update here,
		//  see https://book-v1.book.kubebuilder.io/basics/status_subresource.html
		updateErr = c.cli.Status().Update(ctx, tc)

		if updateErr == nil {
			klog.Infof("tiflow cluster: [%s/%s] updated successfully", ns, tcName)
			return nil
		}
		klog.Infof("failed to update tiflow cluster: [%s/%s], error: %v", ns, tcName, updateErr)

		updated := &v1alpha1.TiflowCluster{}
		if err := c.cli.Get(ctx, types.NamespacedName{
			Namespace: ns,
			Name:      tcName,
		}, updated); err == nil {
			// make a copy, so we don't mutate the shared cache
			tc = updated.DeepCopy()
			tc.Status = *status
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated tiflow cluster %s/%s: %v", ns, tcName, err))
		}

		return updateErr
	})
	if err != nil {
		klog.Errorf("failed to update tiflow cluster: [%s/%s], error: %v", ns, tcName, err)
		// return result.UpdateClusterStatus{Err: err}
		return err
	}

	return nil
}
