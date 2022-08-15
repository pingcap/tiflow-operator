package utils

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/pingcap/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tiflow-operator/api/config"
	pingcapcomv1alpha1 "github.com/pingcap/tiflow-operator/api/v1alpha1"
)

func AddConfigMapDigestSuffix(cm *corev1.ConfigMap) error {
	sum, err := Sha256Sum(cm.Data)
	if err != nil {
		return err
	}
	suffix := fmt.Sprintf("%x", sum)[0:7]
	cm.Name = fmt.Sprintf("%s-%s", cm.Name, suffix)
	return nil
}

func Sha256Sum(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum), nil
}

// FindConfigMapVolume returns the configmap which's name matches the predicate in a PodSpec, empty indicates not found
func FindConfigMapVolume(podSpec *corev1.PodSpec, pred func(string) bool) string {
	for _, vol := range podSpec.Volumes {
		if vol.ConfigMap != nil && pred(vol.ConfigMap.LocalObjectReference.Name) {
			return vol.ConfigMap.LocalObjectReference.Name
		}
	}
	return ""
}

func updateConfigMap(old, new *corev1.ConfigMap) (bool, error) {
	dataEqual := true

	// check config
	tomlField := []string{
		"config-file", // tiflow
	}
	for _, k := range tomlField {
		oldData, oldOK := old.Data[k]
		newData, newOK := new.Data[k]

		if oldOK != newOK {
			dataEqual = false
		}

		if !oldOK || !newOK {
			continue
		}

		equal, err := config.Equal([]byte(oldData), []byte(newData))
		if err != nil {
			return false, errors.Annotatef(err, "compare %s/%s %s and %s failed", old.Namespace, old.Name, oldData, newData)
		}

		if equal {
			new.Data[k] = oldData
		} else {
			dataEqual = false
		}
	}

	// check startup script
	field := "startup-script"
	oldScript, oldExist := old.Data[field]
	newScript, newExist := new.Data[field]
	if oldExist != newExist {
		dataEqual = false
	} else if oldExist && newExist {
		if oldScript != newScript {
			dataEqual = false
		}
	}

	return dataEqual, nil
}

// UpdateConfigMapIfNeed set the toml field as the old one if they are logically equal.
func UpdateConfigMapIfNeed(
	ctx context.Context,
	cli client.Client,
	configUpdateStrategy pingcapcomv1alpha1.ConfigUpdateStrategy,
	inUseName string,
	desired *corev1.ConfigMap,
) error {

	switch configUpdateStrategy {
	case pingcapcomv1alpha1.ConfigUpdateStrategyInPlace:
		if inUseName != "" {
			desired.Name = inUseName
		}
		return nil
	case pingcapcomv1alpha1.ConfigUpdateStrategyRollingUpdate:
		existing := &corev1.ConfigMap{}
		err := cli.Get(ctx, types.NamespacedName{
			Namespace: desired.Namespace,
			Name:      inUseName,
		}, existing)
		if err != nil {
			if errors.IsNotFound(err) {
				AddConfigMapDigestSuffix(desired)
				return nil
			}

			return errors.AddStack(err)
		}

		dataEqual, err := updateConfigMap(existing, desired)
		if err != nil {
			return err
		}

		AddConfigMapDigestSuffix(desired)

		confirmNameByData(existing, desired, dataEqual)

		return nil
	default:
		return errors.Errorf("unknown config update strategy: %v", configUpdateStrategy)

	}
}

// confirmNameByData is used to fix the problem that
// when configUpdateStrategy is changed from InPlace to RollingUpdate for the first time,
// the name of desired configmap maybe different from the existing one while
// the data of them are the same, which will cause rolling update.
func confirmNameByData(existing, desired *corev1.ConfigMap, dataEqual bool) {
	if dataEqual && existing.Name != desired.Name {
		desired.Name = existing.Name
	}
	if !dataEqual && existing.Name == desired.Name {
		desired.Name = fmt.Sprintf("%s-new", desired.Name)
	}
}
