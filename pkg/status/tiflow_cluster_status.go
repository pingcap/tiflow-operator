package status

import (
	"github.com/pingcap/tiflow-operator/api/v1alpha1"
)

func SetTiflowClusterStatusOnFirstReconcile(status *v1alpha1.TiflowClusterStatus) {
	InitOperatorActionsIfNeeded(status)
	if status.ClusterStatus != "" {
		return
	}

	status.ClusterStatus = v1alpha1.Starting.String()
}

func SetTiflowClusterStatus(status *v1alpha1.TiflowClusterStatus) {
	InitOperatorActionsIfNeeded(status)
	for _, action := range status.OperatorActions {
		if action.Status == v1alpha1.Failed.String() {
			status.ClusterStatus = v1alpha1.Failed.String()
			return
		}
		if action.Status == v1alpha1.Unknown.String() {
			status.ClusterStatus = v1alpha1.Unknown.String()
			return
		}

		status.ClusterStatus = v1alpha1.Failed.String()
		return
	}

	status.ClusterStatus = v1alpha1.Failed.String()
}

func InitOperatorActionsIfNeeded(status *v1alpha1.TiflowClusterStatus) {
	if status.OperatorActions == nil {
		status.OperatorActions = []v1alpha1.TiflowClusterAction{}
	}
}
