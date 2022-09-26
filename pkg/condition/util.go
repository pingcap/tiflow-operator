package condition

import (
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// findContainerByName finds targetContainer by containerName, If not find, then return nil
func findContainerByName(sts *apps.StatefulSet, containerName string) *corev1.Container {
	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == containerName {
			return &c
		}
	}
	return nil
}
