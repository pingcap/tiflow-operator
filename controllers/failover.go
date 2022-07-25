package controllers

import "github.com/StepOnce7/tiflow-operator/api/v1alpha1"

// Since the "Unhealthy" is a very universal event reason string, which could apply to all the tiflow cluster components,
// we should make a global event module, and put event related constants there.
const (
	unHealthEventReason     = "Unhealthy"
	unHealthEventMsgPattern = "%s pod[%s] is unhealthy, msg:%s"
	FailedSetStoreLabels    = "FailedSetStoreLabels"
)

// Failover implements the logic for tiflow cluster failover and recovery.
type Failover interface {
	Failover(cluster *v1alpha1.TiflowCluster) error
	Recover(cluster *v1alpha1.TiflowCluster)
	RemoveUndesiredFailures(cluster *v1alpha1.TiflowCluster)
}
