package member

import (
	"github.com/pingcap/tiflow-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type executorUpgrader struct {
	client client.Client
}

// NewExecutorUpgrader returns a executorUpgrader
func NewExecutorUpgrader(cli client.Client) Upgrader {
	return &executorUpgrader{
		client: cli,
	}
}

func (u *executorUpgrader) Upgrade(tc *v1alpha1.TiflowCluster, oldSts *appsv1.StatefulSet, newSts *appsv1.StatefulSet) error {
	return u.gracefulUpgrade(tc, oldSts, newSts)
}

func (u *executorUpgrader) gracefulUpgrade(tc *v1alpha1.TiflowCluster, oldSts, newSts *appsv1.StatefulSet) error {

	return nil
}

func (u *executorUpgrader) upgradeExecutorPod(tc *v1alpha1.TiflowCluster, ordinal int32, newSts *appsv1.StatefulSet) error {
	return nil
}
