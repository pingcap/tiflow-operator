package utils

import (
	"fmt"
	"github.com/StepOnce7/tiflow-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	ControllerKind = schema.GroupVersion{Group: "pingcap.com", Version: "v1alpha1"}.WithKind("TiflowCluster")
)

// TiFlowExecutorMemberName returns tiflow-engine-executor member name
func TiFlowExecutorMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tiflow-executor", clusterName)
}

// TiFlowExecutorPeerMemberName returns tiflow-engine-executor peer service name
func TiFlowExecutorPeerMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tiflow-executor-peer", clusterName)
}

func GetTiFlowOwnerRef(tc *v1alpha1.TiflowCluster) metav1.OwnerReference {
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
