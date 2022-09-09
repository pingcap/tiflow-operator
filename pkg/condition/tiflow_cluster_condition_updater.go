package condition

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	utiltiflowcluster "github.com/pingcap/tiflow-operator/pkg/util/tiflowcluster"
)

// ConditionUpdater interface that translates cluster state into
// into dm cluster status conditions.
type ConditionUpdater interface {
	Update(*v1alpha1.TiflowCluster) error
}

type RealConditionUpdater struct {
}

var _ ConditionUpdater = &RealConditionUpdater{}

func (u *RealConditionUpdater) Update(tc *v1alpha1.TiflowCluster) error {
	u.updateReadyCondition(tc)
	// in the future, we may return error when we need to Kubernetes API, etc.
	return nil
}

func allStatefulSetsAreUpToDate(dc *v1alpha1.TiflowCluster) bool {
	isUpToDate := func(status *appsv1.StatefulSetStatus, requireExist bool) bool {
		if status == nil {
			return !requireExist
		}
		return status.CurrentRevision == status.UpdateRevision
	}
	return (isUpToDate(dc.Status.Master.StatefulSet, true)) &&
		(isUpToDate(dc.Status.Executor.StatefulSet, false))
}

func (u *RealConditionUpdater) updateReadyCondition(tc *v1alpha1.TiflowCluster) {
	status := v1.ConditionFalse
	reason := ""
	message := ""

	switch {
	case !allStatefulSetsAreUpToDate(tc):
		reason = utiltiflowcluster.StatfulSetNotUpToDate
		message = "statefulset(s) are in progress"
	case !tc.AllMasterMembersReady():
		reason = utiltiflowcluster.MasterNotUpYet
		message = "tiflow-master(s) are not up yet"
	case !tc.AllExecutorMembersReady():
		reason = utiltiflowcluster.ExecutorNotUpYet
		message = "tiflow-executor(s) are not up yet"
	default:
		status = v1.ConditionTrue
		reason = utiltiflowcluster.Ready
		message = "tiflow cluster is fully up and running"
	}
	cond := utiltiflowcluster.NewTiflowClusterCondition(utiltiflowcluster.Ready, status, reason, message)
	utiltiflowcluster.SetTiflowClusterCondition(&tc.Status, *cond)
}
