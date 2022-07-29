package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/StepOnce7/tiflow-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	errorsv1 "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UpdateConfigMapIfNeed set the toml field as the old one if they are logically equal.
func UpdateConfigMapIfNeed(
	ctx context.Context,
	client client.Client,
	configUpdateStrategy v1alpha1.ConfigUpdateStrategy,
	inUseName string,
	desired *corev1.ConfigMap) error {

	cm := &corev1.ConfigMap{}
	switch configUpdateStrategy {
	case v1alpha1.ConfigUpdateStrategyInPlace:
		if inUseName != "" {
			desired.Name = inUseName
		}
		return nil
	case v1alpha1.ConfigUpdateStrategyRollingUpdate:
		err := client.Get(ctx, types.NamespacedName{
			Namespace: desired.Namespace,
			Name:      inUseName,
		}, cm)

		if err != nil {
			if errorsv1.IsNotFound(err) {
				AddConfigMapDigestSuffix(desired)
				return nil
			}
			return err
		}

		dataEqual, err := updateConfigMap(cm, desired)
		if err != nil {
			return err
		}
		AddConfigMapDigestSuffix(desired)
		confirmNameByData(cm, desired, dataEqual)

		return nil

	default:
		str := fmt.Sprintf("unknown config update strategy: %v", configUpdateStrategy)
		return errors.New(str)
	}

	return nil
}

func updateConfigMap(old, new *corev1.ConfigMap) (bool, error) {

	return false, nil
}

func confirmNameByData(existing, desired *corev1.ConfigMap, dataEqual bool) {
	if dataEqual && existing.Name != desired.Name {
		desired.Name = existing.Name
	}
	if !dataEqual && existing.Name == desired.Name {
		desired.Name = fmt.Sprintf("%s-new", desired.Name)
	}
}

func FindConfigMaoVolume(podSpec *corev1.PodSpec, pred func(string) bool) string {
	for _, vol := range podSpec.Volumes {
		if vol.ConfigMap != nil && pred(vol.ConfigMap.LocalObjectReference.Name) {
			return vol.ConfigMap.LocalObjectReference.Name
		}
	}
	return ""
}
