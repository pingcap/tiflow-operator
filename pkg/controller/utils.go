package controller

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pingcapcomv1alpha1 "github.com/pingcap/tiflow-operator/api/v1alpha1"
)

var (
	// ControllerKind contains the group version for tiflowcluster controller type.
	ControllerKind = pingcapcomv1alpha1.GroupVersion.WithKind("TiflowCluster")
)

// RequeueError is used to requeue the item, this error type should't be considered as a real error
type RequeueError struct {
	s string
}

func (re *RequeueError) Error() string {
	return re.s
}

// RequeueErrorf returns a RequeueError
func RequeueErrorf(format string, a ...interface{}) error {
	return &RequeueError{fmt.Sprintf(format, a...)}
}

// IsRequeueError returns whether err is a RequeueError
func IsRequeueError(err error) bool {
	_, ok := err.(*RequeueError)
	return ok
}

// TiflowMasterMemberName returns tiflow-master member name
func TiflowMasterMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tiflow-master", clusterName)
}

// TiflowMasterFullHost returns tiflow-master full host
func TiflowMasterFullHost(clusterName, namespace, clusterDomain string) string {
	svc := ""
	if clusterDomain != "" || namespace != "" {
		svc = ".svc"
	}
	if clusterDomain != "" {
		clusterDomain = "." + clusterDomain
	}
	if namespace != "" {
		namespace = "." + namespace
	}
	return fmt.Sprintf("%s-tiflow-master%s%s%s", clusterName, namespace, svc, clusterDomain)
}

// TiflowMasterPeerMemberName returns tiflow-master peer service name
func TiflowMasterPeerMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tiflow-master-peer", clusterName)
}

// TiflowExecutorMemberName returns tiflow-executor member name
func TiflowExecutorMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tiflow-executor", clusterName)
}

// TiflowExecutorPeerMemberName returns tiflow-executor peer service name
func TiflowExecutorPeerMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tiflow-executor-peer", clusterName)
}

// GetOwnerRef returns TiflowCluster's OwnerReference
func GetOwnerRef(tc *pingcapcomv1alpha1.TiflowCluster) metav1.OwnerReference {
	controller := true
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         ControllerKind.GroupVersion().String(),
		Kind:               ControllerKind.Kind,
		Name:               tc.GetName(),
		UID:                tc.GetUID(),
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}

// AnnProm adds annotations for prometheus scraping metrics
func AnnProm(port int32) map[string]string {
	return map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/path":   "/metrics",
		"prometheus.io/port":   fmt.Sprintf("%d", port),
	}
}

func ContainerResource(req corev1.ResourceRequirements) corev1.ResourceRequirements {
	trimmed := req.DeepCopy()
	if trimmed.Limits != nil {
		delete(trimmed.Limits, corev1.ResourceStorage)
	}
	if trimmed.Requests != nil {
		delete(trimmed.Requests, corev1.ResourceStorage)
	}
	return *trimmed
}
