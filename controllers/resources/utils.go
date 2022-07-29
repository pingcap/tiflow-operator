package resources

import (
	"crypto/sha256"
	"fmt"
	"github.com/StepOnce7/tiflow-operator/api/v1alpha1"
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

func AnnotationsMountVolume() (corev1.VolumeMount, corev1.Volume) {
	// todo:
	m := corev1.VolumeMount{Name: "annotation", ReadOnly: true, MountPath: ""}
	v := corev1.Volume{
		Name: "annotation",
		VolumeSource: corev1.VolumeSource{
			DownwardAPI: &corev1.DownwardAPIVolumeSource{
				Items: []corev1.DownwardAPIVolumeFile{
					{
						Path:     "annotation",
						FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations"},
					},
				},
			},
		},
	}

	return m, v
}

func GetStorageVolumeName(storageVolumeName string, memberType v1alpha1.MemberType) v1alpha1.StorageVolumeName {
	if storageVolumeName == "" {
		return v1alpha1.StorageVolumeName(memberType.String())
	}

	return v1alpha1.StorageVolumeName(fmt.Sprintf("%s-%s", memberType.String(), storageVolumeName))
}

func ParseStorageRequest(req corev1.ResourceList) (corev1.ResourceRequirements, error) {
	//TODO implement me
	panic("implement me")
}

func ContainerResource(req corev1.ResourceRequirements) corev1.ResourceRequirements {
	//TODO implement me
	panic("implement me")
}
