package status

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"github.com/pingcap/tiflow-operator/pkg/result"
)

type TiflowClusterStatusManager struct {
	cli            client.Client
	clientSet      kubernetes.Interface
	cluster        *v1alpha1.TiflowCluster
	MasterStatus   SyncPhaseManager
	ExecutorStatus SyncPhaseManager
}

func NewTiflowClusterStatusManager(cli client.Client, clientSet kubernetes.Interface, tc *v1alpha1.TiflowCluster) UpdateStatus {
	return &TiflowClusterStatusManager{
		cli:            cli,
		clientSet:      clientSet,
		cluster:        tc,
		MasterStatus:   NewMasterPhaseManager(cli, clientSet, tc),
		ExecutorStatus: NewExecutorPhaseManager(cli, clientSet, tc),
	}
}

// setTiflowClusterStatusOnFirstReconcile will set phase of TiflowCluster as Starting on reconcile first
func setTiflowClusterStatusOnFirstReconcile(status *v1alpha1.TiflowClusterStatus) {
	if status.ClusterPhase != "" {
		return
	}
	setMasterClusterStatusOnFirstReconcile(&status.Master)
	setExecutorClusterStatusOnFirstReconcile(&status.Executor)
	status.ClusterPhase = v1alpha1.ClusterStarting
	status.Message = "Starting... tiflow-cluster on first reconcile. Just a moment"
	status.LastTransitionTime = metav1.Now()
}

func (tcsm *TiflowClusterStatusManager) SyncTiflowClusterPhase() {
	if tcsm.cluster.Status.ClusterPhase == "" {
		setTiflowClusterStatusOnFirstReconcile(tcsm.cluster.GetClusterStatus())
		return
	}

	tcsm.MasterStatus.SyncPhase()
	tcsm.ExecutorStatus.SyncPhase()

	masterPhase, executorPhase := tcsm.cluster.GetMasterPhase(), tcsm.cluster.GetExecutorPhase()
	switch {
	case masterPhase == v1alpha1.MasterFailed || executorPhase == v1alpha1.ExecutorFailed:
		tcsm.SetTiflowClusterPhase(v1alpha1.ClusterFailed, "errors, failed phase for Master or Executor")
	case masterPhase == v1alpha1.MasterUnknown || executorPhase == v1alpha1.ExecutorUnknown:
		tcsm.SetTiflowClusterPhase(v1alpha1.ClusterUnknown, "errors, unknown phase for Master or Executor")
	case masterPhase == v1alpha1.MasterRunning && executorPhase == v1alpha1.ExecutorRunning:
		tcsm.SetTiflowClusterPhase(v1alpha1.ClusterCompleted, "reconcile completed successfully. Enjoying...")
	default:
		tcsm.SetTiflowClusterPhase(v1alpha1.ClusterReconciling, "reconciling... Just a moment")
	}
}

func (tcsm *TiflowClusterStatusManager) SetTiflowClusterPhase(phase v1alpha1.TiflowClusterPhaseType, message string) {
	clusterStatus := tcsm.cluster.GetClusterStatus()
	clusterStatus.ClusterPhase = phase
	clusterStatus.Message = message
	clusterStatus.LastTransitionTime = metav1.Now()
}

func (tcsm TiflowClusterStatusManager) UpdateTilfowClusterPhase() error {

	ns := tcsm.cluster.GetNamespace()
	tcName := tcsm.cluster.GetName()

	status := tcsm.cluster.Status.DeepCopy()

	// don't wait due to limited number of clients, but backoff after the default number of steps
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var updateErr error
		// tc will be updated in Update function, must use cli.Status().Update instead of cli.Update here,
		//  see https://book-v1.book.kubebuilder.io/basics/status_subresource.html
		updateErr = tcsm.cli.Status().Update(context.TODO(), tcsm.cluster)

		if updateErr == nil {
			klog.Infof("tiflow cluster: [%s/%s] updated successfully", ns, tcName)
			return nil
		}
		klog.Infof("failed to update tiflow cluster: [%s/%s], error: %v", ns, tcName, updateErr)

		updated := &v1alpha1.TiflowCluster{}
		if err := tcsm.cli.Get(context.TODO(), types.NamespacedName{
			Namespace: ns,
			Name:      tcName,
		}, updated); err == nil {
			// make a copy, so we don't mutate the shared cache
			tcsm.cluster = updated.DeepCopy()
			tcsm.cluster.Status = *status
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated tiflow cluster %s/%s: %v", ns, tcName, err))
		}

		return updateErr
	})
	if err != nil {
		// return result.UpdateClusterStatus{Err: err}
		return result.UpdateStatusErr{
			Err: fmt.Errorf("failed to update tiflow cluster: [%s/%s], error: %v", ns, tcName, err),
		}
	}

	return nil
}

func (tcsm *TiflowClusterStatusManager) Update() error {

	tcsm.SyncTiflowClusterPhase()

	if err := tcsm.UpdateTilfowClusterPhase(); err != nil {
		return err
	}

	return nil
}
