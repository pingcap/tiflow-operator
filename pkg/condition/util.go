package condition

import (
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func masterMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tiflow-master", clusterName)
}

func executorMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tiflow-executor", clusterName)
}

// findContainerByName finds targetContainer by containerName, If not find, then return nil
func findContainerByName(sts *appsv1.StatefulSet, containerName string) *corev1.Container {
	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == containerName {
			return &c
		}
	}
	return nil
}

func formatName(name string) (string, error) {
	nameSlice := strings.Split(name, ".")
	if len(nameSlice) != 4 {
		return "", fmt.Errorf("split name %s error", name)
	}

	res := fmt.Sprintf("%s.%s", nameSlice[0], nameSlice[2])
	return res, nil
}

func statefulSetUpToDate(sts *appsv1.StatefulSetStatus, requireExist bool) bool {
	if sts == nil {
		return !requireExist
	}
	return sts.CurrentRevision == sts.UpdateRevision
}
