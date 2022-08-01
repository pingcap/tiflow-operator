package member

import (
	"github.com/StepOnce7/tiflow-operator/api/v1alpha1"
	apps "k8s.io/api/apps/v1"
)

// Upgrader implements the logic for upgrading the tiflow cluster.
type Upgrader interface {
	// Upgrade upgrade the cluster
	Upgrade(*v1alpha1.TiflowCluster, *apps.StatefulSet, *apps.StatefulSet) error
}
