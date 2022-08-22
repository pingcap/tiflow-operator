package prune

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"github.com/pingcap/tiflow-operator/pkg/controller"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// PersistentVolumePruner provides a Prune() method to remove unused statefulset
// PVC and their underlying PVs.
// The underlying PVs SHOULD have their reclaim policy set to delete.
type PersistentVolumePruner struct {
	ClientSet       kubernetes.Interface
	Pruner          bool
	Namespace       string
	InstanceName    string
	StatefulSetName string
}

func NewPersistentVolumePruner(clientSet kubernetes.Interface, tc *v1alpha1.TiflowCluster) PVCPruner {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	stsName := controller.TiflowExecutorMemberName(tcName)
	prune := tc.Spec.Executor.Stateful

	return &PersistentVolumePruner{
		clientSet,
		prune,
		ns,
		tcName,
		stsName,
	}
}

func (p *PersistentVolumePruner) Prune(ctx context.Context) error {
	sts, err := p.ClientSet.AppsV1().StatefulSets(p.Namespace).Get(
		ctx,
		p.StatefulSetName,
		metav1.GetOptions{},
	)
	if err != nil {
		return fmt.Errorf("getting statefulSet error: %v", err)
	}

	if sts.Spec.Replicas == nil {
		return errors.New("statefulSet had nil")
	}

	// todo: Need to be modified
	//ctx, cancel := context.WithCancel(ctx)
	//defer cancel()
	//if err = p.watchStatefulSet(ctx, cancel, sts); err != nil {
	//	return fmt.Errorf("setting up statefulset watcher error: %v", err)
	//}

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

func (p *PersistentVolumePruner) IsStateful() bool {
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

	prefixes := make([]string, len(sts.Spec.VolumeClaimTemplates))
	pvcsToKeep := make(map[string]bool, int(*sts.Spec.Replicas)*len(sts.Spec.VolumeClaimTemplates))

	for i, pvcTemp := range sts.Spec.VolumeClaimTemplates {
		prefixes[i] = fmt.Sprintf("%s-%s", pvcTemp.Name, sts.Name)

		for j := int32(0); j < *sts.Spec.Replicas; j++ {
			name := fmt.Sprintf("%s-%s-%d", pvcTemp.Name, sts.Name, j)
			pvcsToKeep[name] = true
		}
	}

	selector, err := metav1.LabelSelectorAsSelector(sts.Spec.Selector)
	if err != nil {
		return nil, errors.New("converting statefulset selector to metav1 selector")
	}

	pvcs, err := p.ClientSet.CoreV1().PersistentVolumeClaims(p.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, errors.New("listing PVCs to consider deleting")
	}

	i := 0
	for _, pvc := range pvcs.Items {
		if pvcsToKeep[pvc.Name] {
			continue
		}

		matched := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(pvc.Name, prefix) {
				matched = true
				break
			}
		}

		if !matched {
			continue
		}

		pvcs.Items[i] = pvc
		i++
	}

	pvcs.Items = pvcs.Items[:i]

	sort.Slice(pvcs.Items, func(i, j int) bool {
		return pvcs.Items[i].Name < pvcs.Items[j].Name
	})

	return pvcs.Items, nil

}

func (p *PersistentVolumePruner) pvcsToRemove(ctx context.Context, pvcs []corev1.PersistentVolumeClaim) error {
	gracePeriod := int64(60)
	propagationPolicy := metav1.DeletePropagationForeground

	for _, pvc := range pvcs {
		select {
		case <-ctx.Done():
			return errors.New("concurrent statefulSet modification detected")
		default:

		}
		klog.Infof("deleting PVC, Name: %s", pvc.Name)
		if err := p.ClientSet.CoreV1().PersistentVolumeClaims(p.Namespace).Delete(ctx, pvc.Name, metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
			PropagationPolicy:  &propagationPolicy,
			Preconditions: &metav1.Preconditions{
				UID:             &pvc.UID,
				ResourceVersion: &pvc.ResourceVersion,
			},
		}); err != nil {
			return fmt.Errorf("delting PVC %s err, error: %v", pvc.Name, err)
		}

	}

	return nil
}
