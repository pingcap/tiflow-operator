package member

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"github.com/pingcap/tiflow-operator/pkg/condition"
	"github.com/pingcap/tiflow-operator/pkg/manager/member/prune"
	"github.com/pingcap/tiflow-operator/pkg/status"
)

const defaultSleepTime = 10 * time.Second

type executorScaler struct {
	ClientSet kubernetes.Interface
	PVCPruner prune.PVCPruner
}

// NewExecutorScaler return a executorScaler
func NewExecutorScaler(clientSet kubernetes.Interface) Scaler {

	return &executorScaler{
		ClientSet: clientSet,
		PVCPruner: prune.NewPersistentVolumePruner(clientSet),
	}
}

func (s *executorScaler) Scale(meta metav1.Object, oldSts *appsv1.StatefulSet, newSts *appsv1.StatefulSet) error {

	actual := *oldSts.Spec.Replicas
	desired := *newSts.Spec.Replicas

	scaling := desired - actual
	if scaling > 0 {
		return s.ScaleOut(meta, oldSts, newSts)
	} else if scaling < 0 {
		return s.ScaleIn(meta, oldSts, newSts)
	}

	return nil
}

func (s *executorScaler) ScaleOut(meta metav1.Object, actual *appsv1.StatefulSet, desired *appsv1.StatefulSet) (err error) {
	tc, ok := meta.(*v1alpha1.TiflowCluster)
	if !ok {
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()
	stsName := actual.GetName()

	condition.SetFalse(v1alpha1.ExecutorSynced, &tc.Status, metav1.Now())
	syncState := status.NewExecutorSyncTypeManager(&tc.Status.Executor)
	syncState.Ongoing(v1alpha1.ScaleOutType,
		fmt.Sprintf("tiflow executor [%s/%s] sacling out...", ns, tcName))
	defer func() {
		if err != nil {
			syncState.Failed(v1alpha1.ScaleOutType,
				fmt.Sprintf("tiflow executor [%s/%s] scaling out failed", ns, tcName))
		}
	}()

	ctx := context.TODO()
	// skip this logic if Executor is stateful
	if !tc.Spec.Executor.Stateful {
		klog.Infof("tiflow-executor statefulSet %s for [%s/%s], PVC pruning for Scaling Up",
			stsName, ns, tcName)
		if err := s.PVCPruner.Prune(ctx, tc); err != nil {
			return err
		}
	}

	klog.Infof("start to scaling up tiflow-executor statefulSet %s for [%s/%s], actual: %d, desired: %d",
		stsName, ns, tcName, *actual.Spec.Replicas, *desired.Spec.Replicas)

	out := *desired.Spec.Replicas - *actual.Spec.Replicas
	current := *actual.Spec.Replicas

	for i := out; i > 0; i-- {
		klog.Infof("scaling out statefulSet %s of executor, current: %d, desired: %d",
			stsName, current, current+1)

		if err = s.SetReplicas(ctx, actual, uint(current+1)); err != nil {
			return err
		}

		if err = s.WaitUntilRunning(ctx); err != nil {
			return err
		}

		if err = s.WaitUntilHealthy(ctx, uint(current+1)); err != nil {
			return err
		}

		current++
		time.Sleep(defaultSleepTime)
	}

	klog.Infof("scaling out is done, tiflow-executor statefulSet %s for [%s/%s], current: %d, desired: %d",
		stsName, ns, tcName, current, *desired.Spec.Replicas)

	syncState.Complied(v1alpha1.ScaleOutType,
		fmt.Sprintf("tiflow executor [%s/%s] sacling out completed", ns, tcName))

	return nil
}

func (s *executorScaler) ScaleIn(meta metav1.Object, actual *appsv1.StatefulSet, desired *appsv1.StatefulSet) (err error) {
	tc, ok := meta.(*v1alpha1.TiflowCluster)
	if !ok {
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()
	stsName := actual.GetName()

	condition.SetFalse(v1alpha1.ExecutorSynced, &tc.Status, metav1.Now())
	syncState := status.NewExecutorSyncTypeManager(&tc.Status.Executor)
	syncState.Ongoing(v1alpha1.ScaleInType,
		fmt.Sprintf("tiflow executor [%s/%s] sacling in...", ns, tcName))
	defer func() {
		if err != nil {
			syncState.Failed(v1alpha1.ScaleOutType,
				fmt.Sprintf("tiflow executor [%s/%s] scaling in failed", ns, tcName))
		}
	}()

	klog.Infof("start to scaling in tiflow-executor statefulSet %s for [%s/%s], actual: %d, desired: %d",
		stsName, ns, tcName, *actual.Spec.Replicas, *desired.Spec.Replicas)

	in := *actual.Spec.Replicas - *desired.Spec.Replicas
	current := *actual.Spec.Replicas
	ctx := context.TODO()

	for i := in; i > 0; i-- {
		klog.Infof("scaling in statefulSet %s of executor, current: %d, desired: %d",
			stsName, current, current-1)

		if err = s.SetReplicas(ctx, actual, uint(current-1)); err != nil {
			return err
		}

		if err = s.WaitUntilHealthy(ctx, uint(current-1)); err != nil {
			return err
		}

		current--
		time.Sleep(defaultSleepTime)
	}

	klog.Infof("scaling in is done, tiflow-executor statefulSet %s for [%s/%s], current: %d, desired: %d",
		stsName, ns, tcName, current, *desired.Spec.Replicas)

	if !tc.Spec.Executor.Stateful {
		klog.Infof("tiflow-executor statefulSet %s for [%s/%s], PVC pruning for Scaling Down",
			stsName, ns, tcName)
		if err := s.PVCPruner.Prune(ctx, tc); err != nil {
			return err
		}
	}

	syncState.Complied(v1alpha1.ScaleOutType,
		fmt.Sprintf("tiflow executor [%s/%s] sacling in completed", ns, tcName))

	return nil
}

func (s *executorScaler) SetReplicas(ctx context.Context, actual *appsv1.StatefulSet, desired uint) error {
	_, err := s.ClientSet.AppsV1().StatefulSets(actual.Namespace).UpdateScale(ctx, actual.Name, &autoscaling.Scale{
		ObjectMeta: metav1.ObjectMeta{
			Name:      actual.Name,
			Namespace: actual.Namespace,
		},
		Spec: autoscaling.ScaleSpec{
			Replicas: int32(desired),
		},
	}, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update statefulSet %s of executor, error: %v",
			actual.Name, err)
	}

	return nil
}

// WaitUntilRunning blocks until the tiflow-executor statefulset has the expected number of pods running but not necessarily ready
func (s *executorScaler) WaitUntilRunning(ctx context.Context) error {
	// TODO implement me
	// panic("implement me")
	return nil
}

// WaitUntilHealthy blocks until the tiflow-executor stateful set has exactly `prune` healthy replicas.
func (s *executorScaler) WaitUntilHealthy(ctx context.Context, scale uint) error {
	// TODO implement me
	// panic("implement me")
	return nil
}
