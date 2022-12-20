package condition

import (
	"fmt"
	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"regexp"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	extractPodIDRegexStr = "(.*)-([\\d]+)\\.(.*)"
)

var extracPodIDRegex = regexp.MustCompile(extractPodIDRegexStr)

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

func statefulSetUpToDate(sts *appsv1.StatefulSetStatus, requireExist bool) bool {
	if sts == nil {
		return !requireExist
	}
	return sts.CurrentRevision == sts.UpdateRevision
}

// return clusterName, ordinal, namespace
func getOrdinalFromName(name string, memberType v1alpha1.MemberType) (string, int32, string, error) {
	results := extracPodIDRegex.FindStringSubmatch(name)
	if len(results) < 4 {
		return "", 0, "", perrors.Errorf("can't extract pod id from name %s", name)
	}
	ordinalStr := results[2]
	ordinal, err := strconv.ParseInt(ordinalStr, 10, 32)
	if err != nil {
		return "", 0, "", perrors.Annotatef(err, "fail to parse ordinal %s", ordinalStr)
	}
	return strings.TrimSuffix(results[1], "-"+memberType.String()), int32(ordinal), results[3], nil
}
