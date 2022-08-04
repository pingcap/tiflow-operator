package utils

import (
	"fmt"
	"github.com/StepOnce7/tiflow-operator/api/v1alpha1"
)

func GetStorageVolumeName(storageVolumeName string, memberType v1alpha1.MemberType) v1alpha1.StorageVolumeName {
	if storageVolumeName == "" {
		return v1alpha1.StorageVolumeName(memberType.String())
	}

	return v1alpha1.StorageVolumeName(fmt.Sprintf("%s-%s", memberType.String(), storageVolumeName))
}
