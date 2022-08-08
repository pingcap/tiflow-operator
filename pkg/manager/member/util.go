package member

import (
	"context"
	"encoding/json"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"github.com/pingcap/tiflow-operator/pkg/label"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	LastAppliedConfigAnnotation = "pingcap.com/last-applied-configuration"
)

func getNodePort(svc *v1alpha1.ServiceSpec) int32 {
	if svc.NodePort != nil {
		return *svc.NodePort
	}
	return 0
}

// TODO: check whether do we need this func
// getStsAnnotations gets annotations for statefulset of given component.
//func getStsAnnotations(tcAnns map[string]string, component string) map[string]string {
//	anns := map[string]string{}
//	if tcAnns == nil {
//		return anns
//	}
//
//	// ensure the delete-slots annotation
//	var key string
//	switch component {
//	case label.TiflowMasterLabelVal:
//		key = label.AnnTiflowMasterDeleteSlots
//	case label.TiflowExecutorLabelVal:
//		key = label.AnnTiflowExecutorDeleteSlots
//	default:
//		return anns
//	}
//	if val, ok := tcAnns[key]; ok {
//		anns[helper.DeleteSlotsAnn] = val
//	}
//
//	return anns
//}

func annotationsMountVolume() (corev1.VolumeMount, corev1.Volume) {
	m := corev1.VolumeMount{Name: "annotations", ReadOnly: true, MountPath: "/etc/podinfo"}
	v := corev1.Volume{
		Name: "annotations",
		VolumeSource: corev1.VolumeSource{
			DownwardAPI: &corev1.DownwardAPIVolumeSource{
				Items: []corev1.DownwardAPIVolumeFile{
					{
						Path:     "annotations",
						FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations"},
					},
				},
			},
		},
	}
	return m, v
}

// NeedForceUpgrade check if force upgrade is necessary
func NeedForceUpgrade(ann map[string]string) bool {
	// Check if annotation 'pingcap.com/force-upgrade: "true"' is set
	if ann != nil {
		forceVal, ok := ann[label.AnnForceUpgradeKey]
		if ok && (forceVal == label.AnnForceUpgradeVal) {
			return true
		}
	}
	return false
}

// templateEqual compares the new podTemplateSpec's spec with old podTemplateSpec's last applied config
func templateEqual(new *apps.StatefulSet, old *apps.StatefulSet) bool {
	oldStsSpec := apps.StatefulSetSpec{}
	lastAppliedConfig, ok := old.Annotations[LastAppliedConfigAnnotation]
	if ok {
		err := json.Unmarshal([]byte(lastAppliedConfig), &oldStsSpec)
		if err != nil {
			klog.Errorf("unmarshal PodTemplate: [%s/%s]'s applied config failed,error: %v", old.GetNamespace(), old.GetName(), err)
			return false
		}
		return apiequality.Semantic.DeepEqual(oldStsSpec.Template.Spec, new.Spec.Template.Spec)
	}
	return false
}

// MergeFn knows how to merge a desired object into the current object.
type MergeFn func(existing, desired client.Object) error

func DeepCopyClientObject(input client.Object) client.Object {
	robj := input.DeepCopyObject()
	cobj := robj.(client.Object)
	return cobj
}

func createOrUpdateObject(ctx context.Context, cli client.Client, obj client.Object, mergeFn MergeFn) (runtime.Object, error) {
	// controller-runtime/client will mutate the object pointer in-place,
	// to be consistent with other methods in our controller, we copy the object
	// to avoid the in-place mutation here and hereafter.
	desired := DeepCopyClientObject(obj)

	// 1. try to create and see if there is any conflicts
	err := cli.Create(ctx, desired)
	if errors.IsAlreadyExists(err) {

		// 2. object has already existed, merge our desired changes to it
		existing := DeepCopyClientObject(obj)
		key := client.ObjectKeyFromObject(existing)
		err = cli.Get(ctx, key, desired)
		if err != nil {
			return nil, err
		}

		mutated := DeepCopyClientObject(existing)
		// 4. invoke mergeFn to mutate a copy of the existing object
		if err := mergeFn(mutated, desired); err != nil {
			return nil, err
		}

		// 5. check if the copy is actually mutated
		if !apiequality.Semantic.DeepEqual(existing, mutated) {
			err := cli.Update(ctx, mutated)
			return mutated, err
		}

		return mutated, nil
	}

	return desired, err
}

func mergeConfigMapFunc(existing, desired client.Object) error {
	existingCm := existing.(*corev1.ConfigMap)
	desiredCm := desired.(*corev1.ConfigMap)

	existingCm.Data = desiredCm.Data
	existingCm.Labels = desiredCm.Labels
	for k, v := range desiredCm.Annotations {
		existingCm.Annotations[k] = v
	}
	return nil
}

func getStsAnnotations(tcAnns map[string]string, component string) map[string]string {
	//TODO implement me
	panic("implement me")
}
