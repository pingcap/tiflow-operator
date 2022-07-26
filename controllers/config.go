package controllers

import (
	"github.com/StepOnce7/tiflow-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
)

// UpdateConfigMapIfNeed set the toml field as the old one if they are logically equal.
func UpdateConfigMapIfNeed(
	cmLister corelisterv1.ConfigMapLister,
	configUpdateStrategy v1alpha1.ConfigUpdateStrategy,
	inUseName string,
	desired *corev1.ConfigMap) error {

	return nil
}
