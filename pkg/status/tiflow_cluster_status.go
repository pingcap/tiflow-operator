package status

import "github.com/pingcap/tiflow-operator/api/v1alpha1"

func SetTiflowClusterStatusOnFirstReconcile(status *v1alpha1.TiflowClusterStatus) {
	if status.ClusterStatus != "" {
		return
	}

	status.ClusterStatus = v1alpha1.Starting.String()
}

func SetTiflowClusterStatus(status *v1alpha1.TiflowClusterStatus) {
	panic("not implemented")
}
