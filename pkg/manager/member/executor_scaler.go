package member

import (
	"context"
	"github.com/pingcap/tiflow-operator/pkg/controller"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"github.com/pingcap/tiflow-operator/pkg/manager/member/scale"
)

type executorScaler struct {
	Client    client.Client
	PVCPruner PVCPruner
}

// NewExecutorScaler return a executorScaler
func NewExecutorScaler(cli client.Client, prune bool, tc v1alpha1.TiflowCluster) Scaler {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	stsName := controller.TiflowExecutorMemberName(tcName)

	return &executorScaler{
		Client:    cli,
		PVCPruner: scale.NewPersistentVolumePruner(cli, prune, ns, tcName, stsName),
	}
}

func (s *executorScaler) Scale(ctx context.Context, meta metav1.Object, oldSts *appsv1.StatefulSet, newSts *appsv1.StatefulSet) error {

	actual := *oldSts.Spec.Replicas
	desired := *newSts.Spec.Replicas

	klog.Infof("start scaling logic, desired: %d, actual: %d",
		desired, actual)

	scaling := desired - actual
	if scaling > 0 {
		return s.ScaleOut(ctx, meta, oldSts, newSts)
	} else if scaling < 0 {
		return s.ScaleIn(ctx, meta, oldSts, newSts)
	}

	return nil
}

func (s *executorScaler) ScaleOut(ctx context.Context, meta metav1.Object, actual *appsv1.StatefulSet, desired *appsv1.StatefulSet) error {
	tc, ok := meta.(*v1alpha1.TiflowCluster)
	if !ok {
		return nil
	}

	if s.PVCPruner.IsPrune() {
		klog.Info("PVC pruning for Scaling Up")
		if err := s.PVCPruner.Prune(ctx); err != nil {
			return err
		}
	} else {
		klog.Info("Scaler will not delete the PVC. Please enable prune as true")
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()
	stsName := actual.GetName()

	klog.Infof("start to scaling up tiflow-executor statefulSet %s for [%s/%s]",
		stsName, ns, tcName)

	up := *desired.Spec.Replicas - *actual.Spec.Replicas
	current := *actual.Spec.Replicas

	for i := up; i > 0; i-- {
		klog.Infof("scaling up statefulSet %s, current: %d, desired: %d",
			stsName, current, current+1)

		if err := s.SetReplicas(ctx, actual, uint(current+1)); err != nil {
			return err
		}

		if err := s.WaitUntilRunning(ctx); err != nil {
			return err
		}

		if err := s.WaitUntilHealthy(ctx, uint(current+1)); err != nil {
			return err
		}

		current++
	}

	klog.Infof("scaling up is done, tiflow-executor statefulSet %s for [%s/%s]",
		stsName, ns, tcName)

	klog.Infof("scaling up statefulSet %s, current: %d, desired: %d",
		stsName, current, *desired.Spec.Replicas)

	return nil
}

func (s *executorScaler) ScaleIn(ctx context.Context, meta metav1.Object, actual *appsv1.StatefulSet, desired *appsv1.StatefulSet) error {
	tc, ok := meta.(*v1alpha1.TiflowCluster)
	if !ok {
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()
	stsName := actual.GetName()

	klog.Infof("start to scaling down tiflow-executor statefulSet %s for [%s/%s]",
		stsName, ns, tcName)

	down := *actual.Spec.Replicas - *desired.Spec.Replicas
	current := *actual.Spec.Replicas

	for i := down; i > 0; i-- {
		klog.Infof("scaling down statefulSet %s, current: %d, desired: %d",
			stsName, current, current-1)

		if err := s.SetReplicas(ctx, actual, uint(current-1)); err != nil {
			return err
		}

		if err := s.WaitUntilHealthy(ctx, uint(current-1)); err != nil {
			return err
		}

		current--
	}

	klog.Infof("scaling down is done, tiflow-executor statefulSet %s for [%s/%s]",
		stsName, ns, tcName)

	klog.Infof("scaling up statefulSet %s, current: %d, desired: %d",
		stsName, current, *desired.Spec.Replicas)

	if s.PVCPruner.IsPrune() {
		klog.Info("PVC pruning for Scaling Down")
		if err := s.PVCPruner.Prune(ctx); err != nil {
			return err
		}
	} else {
		klog.Info("Scaler will not delete the PVC. Please enable prune as true")
	}

	return nil
}

func (s *executorScaler) SetReplicas(ctx context.Context, actual *appsv1.StatefulSet, desired uint) error {
	//TODO implement me
	panic("implement me")
}

// WaitUntilRunning blocks until the tiflow-executor statefulset has the expected number of pods running but not necessarily ready
func (s *executorScaler) WaitUntilRunning(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

// WaitUntilHealthy blocks until the tiflow-executor stateful set has exactly `scale` healthy replicas.
func (s *executorScaler) WaitUntilHealthy(ctx context.Context, scale uint) error {
	//TODO implement me
	panic("implement me")
}
