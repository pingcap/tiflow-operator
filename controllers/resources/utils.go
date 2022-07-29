package resources

import (
	"crypto/sha256"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
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
