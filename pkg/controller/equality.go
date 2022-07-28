package controller

import (
	"encoding/json"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/klog/v2"
)

const (
	// LastAppliedPodTemplate is annotation key of the last applied pod template
	LastAppliedPodTemplate = "pingcap.com/last-applied-podtemplate"

	// LastAppliedConfigAnnotation is annotation key of last applied configuration
	LastAppliedConfigAnnotation = "pingcap.com/last-applied-configuration"
)

// ServiceEqual compares the new Service's spec with old Service's last applied config
func ServiceEqual(newSvc, oldSvc *corev1.Service) (bool, error) {
	oldSpec := corev1.ServiceSpec{}
	if lastAppliedConfig, ok := oldSvc.Annotations[LastAppliedConfigAnnotation]; ok {
		err := json.Unmarshal([]byte(lastAppliedConfig), &oldSpec)
		if err != nil {
			klog.Errorf("unmarshal ServiceSpec: [%s/%s]'s applied config failed,error: %v", oldSvc.GetNamespace(), oldSvc.GetName(), err)
			return false, err
		}
		equal := apiequality.Semantic.DeepEqual(oldSpec, newSvc.Spec)
		if !equal {
			if klog.V(2).Enabled() {
				diff := cmp.Diff(oldSpec, newSvc.Spec)
				klog.V(2).Infof("Service spec diff for %s/%s: %s", newSvc.Namespace, newSvc.Name, diff)
			}
		}

		return equal, nil
	}
	return false, nil
}

// SetServiceLastAppliedConfigAnnotation set last applied config info to Service's annotation
func SetServiceLastAppliedConfigAnnotation(svc *corev1.Service) error {
	b, err := json.Marshal(svc.Spec)
	if err != nil {
		return err
	}
	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}
	svc.Annotations[LastAppliedConfigAnnotation] = string(b)
	return nil
}
