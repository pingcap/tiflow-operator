package member

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"github.com/pingcap/tiflow-operator/pkg/manager/member/scale"
)

type executorScaler struct {
	Client    client.Client
	Prune     bool
	PVCPruner PVCPruner
}

func NewExecutorScaler(cli client.Client, prune bool, tc v1alpha1.TiflowCluster) Scaler {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	return &executorScaler{
		Client:    cli,
		Prune:     prune,
		PVCPruner: scale.NewPersistentVolumePruner(cli, ns, tcName),
	}
}

func (s *executorScaler) Scale(meta metav1.Object, oldSts *appsv1.StatefulSet, newSts *appsv1.StatefulSet) error {

	actual := *oldSts.Spec.Replicas
	desired := *newSts.Spec.Replicas

	klog.Infof("start scaling logic, desired: %d, actual: %d",
		desired, actual)

	// todo: Need to complete the logic of Prune
	if s.Prune {
		if err := s.PVCPruner.Prune(context.TODO()); err != nil {
			klog.Info("initial PVC pruning")
			return err
		}
	} else {
		klog.Info("Scaler will not delete the PVC. Please enable prune as true")
	}

	scaling := desired - actual
	if scaling > 0 {
		return s.ScaleOut(meta, oldSts, newSts)
	} else if scaling < 0 {
		return s.ScaleIn(meta, oldSts, newSts)
	}

	if s.Prune {
		if err := s.PVCPruner.Prune(context.TODO()); err != nil {
			klog.Info("final PVC pruning")
			return err
		}
	} else {
		klog.Info("Scaler will not delete the PVC. Please set prune as true")
	}

	return nil
}

func (s *executorScaler) ScaleOut(meta metav1.Object, actual *appsv1.StatefulSet, desired *appsv1.StatefulSet) error {
	_, ok := meta.(*v1alpha1.TiflowCluster)
	if !ok {
		return nil
	}
	return nil
}

func (s *executorScaler) ScaleIn(meta metav1.Object, actual *appsv1.StatefulSet, desired *appsv1.StatefulSet) error {
	//TODO implement me
	panic("implement me")
}
