package status

import (
	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	"k8s.io/client-go/kubernetes"
)

type SyncTypeManager interface {
	UpdateStatus
	Ongoing(v1alpha1.SyncTypeName, string)
	Complied(v1alpha1.SyncTypeName, string)
	Failed(v1alpha1.SyncTypeName, string)
	Unknown(v1alpha1.SyncTypeName, string)
}

type UpdateStatus interface {
	UpdateClusterStatus(clientSet kubernetes.Interface, tc *v1alpha1.TiflowCluster)
}
