package member

import (
	"context"
	"fmt"
	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"github.com/pingcap/tiflow-operator/pkg/controller"
	mngerutils "github.com/pingcap/tiflow-operator/pkg/manager/utils"
	"github.com/pingcap/tiflow-operator/pkg/status"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type executorUpgrader struct {
	client client.Client
}

// NewExecutorUpgrader returns a executorUpgrader
func NewExecutorUpgrader(cli client.Client) Upgrader {
	return &executorUpgrader{
		client: cli,
	}
}

func (u *executorUpgrader) Upgrade(tc *v1alpha1.TiflowCluster, oldSts *appsv1.StatefulSet, newSts *appsv1.StatefulSet) error {
	return u.gracefulUpgrade(tc, oldSts, newSts)
}

func (u *executorUpgrader) gracefulUpgrade(tc *v1alpha1.TiflowCluster, oldSts, newSts *appsv1.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	state := status.NewExecutorSyncTypeManager(&tc.Status.Executor)
	state.SetClusterSyncTypeOngoing(v1alpha1.UpgradeType,
		fmt.Sprintf("tiflow executor [%s/%s] upgrading...", ns, tcName))

	if !tc.Status.Executor.Synced {
		return fmt.Errorf("tiflowCluster: [%s/%s]'s tiflow-executor status sync failed,"+
			"can not to be upgraded", ns, tcName)
	}

	if tc.ExecutorScaling() {
		klog.Infof("TiflowCluster: [%s/%s]'s tiflow-executor is scaling, can not upgrade tiflow-executor",
			ns, tcName)
		_, podSpec, err := GetLastAppliedConfig(oldSts)
		if err != nil {
			return err
		}
		newSts.Spec.Template.Spec = *podSpec
		return nil
	}

	tc.Status.Executor.Phase = v1alpha1.ExecutorUpgrading
	if !templateEqual(newSts, oldSts) {
		return nil
	}

	if tc.Status.Executor.StatefulSet.UpdateRevision == tc.Status.Executor.StatefulSet.CurrentRevision {
		return nil
	}

	if oldSts.Spec.UpdateStrategy.Type == appsv1.OnDeleteStatefulSetStrategyType ||
		oldSts.Spec.UpdateStrategy.RollingUpdate == nil {
		newSts.Spec.UpdateStrategy = oldSts.Spec.UpdateStrategy
		klog.Warningf("tiflowCluster: [%s/%s] tiflow-executor statefulSet %s UpdateStrategy has been modified manually",
			ns, tcName, oldSts.GetName())
		return nil
	}

	mngerutils.SetUpgradePartition(newSts, *oldSts.Spec.UpdateStrategy.RollingUpdate.Partition)
	for i := *oldSts.Spec.Replicas - 1; i >= 0; i-- {
		podName := TiflowExecutorPodName(tcName, i)
		pod := &corev1.Pod{}
		err := u.client.Get(context.TODO(), types.NamespacedName{
			Namespace: ns,
			Name:      podName,
		}, pod)
		if err != nil {
			return fmt.Errorf("gracefulUpgrade: failed to get pods %s for cluster [%s/%s], error: %s", podName, ns, tcName, err)
		}

		revision, exists := pod.Labels[appsv1.ControllerRevisionHashLabelKey]
		if !exists {
			return controller.RequeueErrorf("tiflowCluster: [%s/%s]'s tiflow-executor pod: [%s] has no label: %s",
				ns, tcName, podName, appsv1.ControllerRevisionHashLabelKey)
		}

		if revision == tc.Status.Executor.StatefulSet.UpdateRevision {
			if !podutil.IsPodReady(pod) {
				return controller.RequeueErrorf("tiflowCluster: [%s/%s]'s upgrade tiflow-executor pod: [%s] is not ready",
					ns, tcName, podName)
			}
			// todo: Need to be modified
			if _, exist := tc.Status.Executor.Members[podName]; !exist {
				return controller.RequeueErrorf("tiflowCluster: [%s/%s]'s upgrade tiflow-executor pod: [%s] is not exist",
					ns, tcName, podName)
			}
			continue
		}
		// todo: Need to re-arrange this executor's tasks in the future
		mngerutils.SetUpgradePartition(newSts, i)
		return nil
	}

	state.SetClusterSyncTypeComplied(v1alpha1.UpgradeType,
		fmt.Sprintf("tiflow executor [%s/%s] completed", ns, tcName))

	return nil
}

// todo: Need to be removed
func (u *executorUpgrader) upgradeExecutorPod(ctx context.Context, tc *v1alpha1.TiflowCluster, ordinal int32, newSts *appsv1.StatefulSet) error {

	mngerutils.SetUpgradePartition(newSts, ordinal)

	return nil
}
