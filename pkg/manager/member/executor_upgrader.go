package member

import (
	"context"
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"github.com/pingcap/tiflow-operator/pkg/condition"
	"github.com/pingcap/tiflow-operator/pkg/controller"
	mngerutils "github.com/pingcap/tiflow-operator/pkg/manager/utils"
	"github.com/pingcap/tiflow-operator/pkg/status"
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

	klog.Infof("start to upgrade tiflow executor [%s/%s]", ns, tcName)

	condition.SetFalse(v1alpha1.ExecutorSyncChecked, tc.GetClusterStatus(), metav1.Now())
	status.Ongoing(v1alpha1.UpgradeType, tc.GetClusterStatus(), v1alpha1.TiFlowExecutorMemberType,
		fmt.Sprintf("tiflow executor [%s/%s] upgrading...", ns, tcName))

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
	if !reflect.DeepEqual(tc.Status.Executor.NodeSelector, tc.Spec.Executor.NodeSelector) {
		tc.Status.Executor.NodeSelector = tc.Spec.Executor.NodeSelector
	}
	if !templateEqual(newSts, oldSts) {
		return nil
	}

	klog.Infof("CurrentRevision: %s, UpdateRevision: %s", tc.Status.Executor.StatefulSet.CurrentRevision, tc.Status.Executor.StatefulSet.UpdateRevision)
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
	klog.Infof("Upgrading tiflow-executor statefulSet")
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
			if _, exist := tc.Status.Executor.Members[podName+"."+ns]; !exist {
				return controller.RequeueErrorf("tiflowCluster: [%s/%s]'s upgrade tiflow-executor pod: [%s] is not exist",
					ns, tcName, podName)
			}
			continue
		}
		// todo: Need to re-arrange this executor's tasks in the future
		mngerutils.SetUpgradePartition(newSts, i)
		return nil
	}

	return nil
}
