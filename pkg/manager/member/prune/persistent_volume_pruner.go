package prune

import (
	"context"
	"errors"
	"fmt"
	"github.com/pingcap/tiflow-operator/pkg/controller"
	"k8s.io/apimachinery/pkg/watch"
	"sort"
	"strings"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
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
	ClientSet kubernetes.Interface
}

// NewPersistentVolumePruner return a new PersistentVolumePruner
func NewPersistentVolumePruner(clientSet kubernetes.Interface) PVCPruner {

	return &PersistentVolumePruner{
		clientSet,
	}
}

// Prune locates and removes all PVCs that belong to a given statefulSet but are not in use.
// The underlying PVs' reclaim policy should be set to delete, other options may result in
// leaking volumes which cost us money.
func (p *PersistentVolumePruner) Prune(ctx context.Context, tc *v1alpha1.TiflowCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	stsName := controller.TiflowExecutorMemberName(tcName)

	sts, err := p.ClientSet.AppsV1().StatefulSets(ns).Get(
		ctx,
		stsName,
		metav1.GetOptions{},
	)
	if err != nil {
		return fmt.Errorf("getting statefulSet error: %v", err)
	}

	if sts.Spec.Replicas == nil {
		return errors.New("statefulSet had nil")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// watchStatefulSet will cancel our ctx if it detects any modifications to
	// sts.Spec.Replicas OR on any unexpected events, namely Deletions.
	if err = p.watchStatefulSet(ctx, cancel, tc, sts); err != nil {
		return fmt.Errorf("setting up statefulset watcher error: %v", err)
	}

	pvcs, err := p.findPVCDeletable(ctx, tc, sts)
	if err != nil {
		return err
	} else if len(pvcs) == 0 {
		klog.Info("No PVC need to prune")
		return nil
	}

	klog.Infof("PVC pruning for [%s/%s], statefulSet: %s, PVCs: %v",
		ns, tcName, stsName, pvcs)

	if err = p.pvcsToRemove(ctx, tc, pvcs); err != nil {
		return fmt.Errorf("pvc remove error: %v", err)
	}

	return nil
}

// watchStatefulSet establishing a watch on the given statefulset of Executor in a
// goroutine and will call the provided cancel function whenever a modification
// to the .Spec.Replicas field OR an unexpected (non-modification) event is
// observed.
func (p *PersistentVolumePruner) watchStatefulSet(
	ctx context.Context,
	cancel context.CancelFunc,
	tc *v1alpha1.TiflowCluster,
	sts *appsv1.StatefulSet,
) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	wacther, err := p.ClientSet.AppsV1().StatefulSets(ns).Watch(ctx, metav1.SingleObject(sts.ObjectMeta))
	if err != nil {
		return fmt.Errorf("establishing watch on statefulSet %s for [%s/%s] error, err: %v",
			sts.Name, ns, tcName, err)
	}

	klog.Infof("establishing watcher on statefulSet %s for [%s/%s]",
		sts.Name, ns, tcName)

	go func() {
		defer wacther.Stop()

		for {
			// For test only
			select {
			case <-ctx.Done():
				return
			default:
			}

			select {
			// For this, we will prevent goroutines from leaking when cancel() has been called.
			case <-ctx.Done():
				return
			case event := <-wacther.ResultChan():
				switch event.Type {
				case watch.Modified:
					if modified, ok := event.Object.(*appsv1.StatefulSet); ok {
						if modified.Spec.Replicas == nil || *modified.Spec.Replicas != *sts.Spec.Replicas {
							cancel()
						}
					}
				default:
					klog.Errorf("saw an unexpected event: %v", event)
					cancel()
				}
			}
		}
	}()

	return nil
}

// findPVCDeletable find all PVCs that were provisioned for the given statefulSet of Executor
// but are not currently in use.
// Use is defined as having an ordinal that is less than the number of expected replicas for get given statefulSet.
func (p *PersistentVolumePruner) findPVCDeletable(ctx context.Context, tc *v1alpha1.TiflowCluster, sts *appsv1.StatefulSet) ([]corev1.PersistentVolumeClaim, error) {
	// K8s doesn't provide a way to tell if a PVC or PV is currently in use by a pod.
	// However, we assume that any PVCs with an ordinal greater than or equal
	// to the sts' Replicas is not in use. As only pods with an ordinal < Replicas will
	// exist. And any PVCs with an ordinal less than Replicas is in use. To detect this, we
	// build a map of PVCs that we consider to be in use and skip and PVCs that it contains
	// the name of.
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

	pvcs, err := p.ClientSet.CoreV1().PersistentVolumeClaims(tc.GetNamespace()).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, errors.New("listing PVCs to consider deleting")
	}

	i := 0
	for _, pvc := range pvcs.Items {
		// Skip if pvc that are still in use.
		if pvcsToKeep[pvc.Name] {
			continue
		}

		// Ensure that any PVC we consider deleting matches the expected naming
		// convention for PVCs managed by a statefulSet.
		// <defaultPVCName>-<stsName>-<ordinal>
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

	// Retain the PVCs that need to be deleted
	pvcs.Items = pvcs.Items[:i]

	// Sort PVCs to ensure we're deleting from lowest to highest.
	// If a new replica is created while we're pruning, and we can
	// not detect it or detect it fast enough, we'll _hopefully_ the
	// requested PVCs will be deleting forcing a new one to be created.
	sort.Slice(pvcs.Items, func(i, j int) bool {
		return pvcs.Items[i].Name < pvcs.Items[j].Name
	})

	return pvcs.Items, nil

}

// pvcsToRemove delete all PVCs that are not currently in use.
func (p *PersistentVolumePruner) pvcsToRemove(ctx context.Context, tc *v1alpha1.TiflowCluster, pvcs []corev1.PersistentVolumeClaim) error {
	gracePeriod := int64(60)
	propagationPolicy := metav1.DeletePropagationForeground

	for _, pvc := range pvcs {
		// For this, ensure that our context is still active.
		// It will be canceled if a change to sts.Spec.Replicas is detected.
		select {
		case <-ctx.Done():
			return errors.New("concurrent statefulSet modification detected")
		default:

		}
		klog.Infof("deleting PVC, Name: %s", pvc.Name)
		if err := p.ClientSet.CoreV1().PersistentVolumeClaims(tc.GetNamespace()).Delete(ctx, pvc.Name, metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
			// Wait for the underlying PV to be deleted before moving on to
			// the next volume.
			PropagationPolicy: &propagationPolicy,
			Preconditions: &metav1.Preconditions{
				// Ensure that this PVC is the same PVC that we slated for
				// deletion.
				UID: &pvc.UID,
				// Ensure that this PVC has not changed since we fetched it.
				// Actually, this not help.
				ResourceVersion: &pvc.ResourceVersion,
			},
		}); err != nil {
			return fmt.Errorf("delting PVC %s err, error: %v", pvc.Name, err)
		}

	}

	return nil
}
