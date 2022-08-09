package tiflowcluster

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"

	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	utiltiflowcluster "github.com/pingcap/tiflow-operator/pkg/util/tiflowcluster"
)

// TiflowClusterConditionUpdater interface that translates cluster state into
// into dm cluster status conditions.
type TiflowClusterConditionUpdater interface {
	Update(*v1alpha1.TiflowCluster) error
}

type tiflowClusterConditionUpdater struct {
}

var _ TiflowClusterConditionUpdater = &tiflowClusterConditionUpdater{}

func (u *tiflowClusterConditionUpdater) Update(tc *v1alpha1.TiflowCluster) error {
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

func (u *tiflowClusterConditionUpdater) updateReadyCondition(tc *v1alpha1.TiflowCluster) {
	status := v1.ConditionFalse
	reason := ""
	message := ""

	switch {
	case !allStatefulSetsAreUpToDate(tc):
		reason = utiltiflowcluster.StatfulSetNotUpToDate
		message = "Statefulset(s) are in progress"
	case !tc.MasterAllMembersReady():
		reason = utiltiflowcluster.MasterUnhealthy
		message = "tiflow-master(s) are not healthy"
	case !tc.ExecutorAllMembersReady():
		reason = utiltiflowcluster.MasterUnhealthy
		message = "some tiflow-worker(s) are not up yet"
	default:
		status = v1.ConditionTrue
		reason = utiltiflowcluster.Ready
		message = "DM cluster is fully up and running"
	}
	cond := utiltiflowcluster.NewTiflowClusterCondition(utiltiflowcluster.Ready, status, reason, message)
	utiltiflowcluster.SetTiflowClusterCondition(&tc.Status, *cond)
}
