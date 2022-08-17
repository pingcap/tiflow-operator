package scale

import (
	"context"
	"fmt"
	"github.com/pingcap/tiflow-operator/pkg/manager/member"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PersistentVolumePruner provides a Prune() method to remove unused statefulset
// PVC and their underlying PVs.
// The underlying PVs SHOULD have their reclaim policy set to delete.
type PersistentVolumePruner struct {
	Client          client.Client
	Pruner          bool
	Namespace       string
	InstanceName    string
	StatefulSetName string
}

func NewPersistentVolumePruner(cli client.Client, prune bool, namespace string, tcName string, stsName string) member.PVCPruner {
	return &PersistentVolumePruner{
		cli,
		prune,
		namespace,
		tcName,
		stsName,
	}
}

func (p *PersistentVolumePruner) Prune(ctx context.Context) error {
	sts := &appsv1.StatefulSet{}
	err := p.Client.Get(ctx, types.NamespacedName{
		Namespace: p.Namespace,
		Name:      p.StatefulSetName,
	}, sts)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := p.watchStatefulSet(ctx, cancel, sts); err != nil {
		return fmt.Errorf("setting up statefulset watcher error: %v", err)
	}

	pvcs, err := p.findPVCDeletable(ctx, sts)
	if err != nil {
		return err
	} else if len(pvcs) == 0 {
		klog.Info("No PVC need to prune")
		return nil
	}

	klog.Infof("PVC pruning for [%s/%s], statefulSet: %s, PVCs: %v",
		p.Namespace, p.InstanceName, p.StatefulSetName, pvcs)

	if err = p.pvcsToRemove(ctx, pvcs); err != nil {
		return fmt.Errorf("pvc remove error: %v", err)
	}

	return nil
}

func (p *PersistentVolumePruner) IsPrune() bool {
	return p.Pruner
}

func (p *PersistentVolumePruner) watchStatefulSet(
	ctx context.Context,
	cancel context.CancelFunc,
	sts *appsv1.StatefulSet,
) error {
	//TODO implement me
	panic("implement me")
}

func (p *PersistentVolumePruner) findPVCDeletable(ctx context.Context, sts *appsv1.StatefulSet) ([]corev1.PersistentVolumeClaim, error) {
	//TODO implement me
	panic("implement me")
}

func (p *PersistentVolumePruner) pvcsToRemove(ctx context.Context, pvcs []corev1.PersistentVolumeClaim) error {
	//TODO implement me
	panic("implement me")
}
