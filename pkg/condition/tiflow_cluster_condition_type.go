package condition

type TiflowClusterConditionType string

const (
	ClusterInitializedCondition TiflowClusterConditionType = "tiflow-cluster-initialized"
	ClusterReadyCondition       TiflowClusterConditionType = "tiflow-cluster-Ready"

	MasterReadyCondition        TiflowClusterConditionType = "tiflow-cluster-master-Ready"
	MasterUpgradeCondition      TiflowClusterConditionType = "tiflow-cluster-master-Upgrade"
	MasterScaleOutCondition     TiflowClusterConditionType = "tiflow-cluster-master-ScaleOut"
	MasterScaleInCondition      TiflowClusterConditionType = "tiflow-cluster-master-ScaleIn"
	MasterDecommissionCondition TiflowClusterConditionType = "tiflow-cluster-master-Decommission"

	ExecutorReadyCondition        TiflowClusterConditionType = "tiflow-cluster-executor-Ready"
	ExecutorUpgradeCondition      TiflowClusterConditionType = "tiflow-cluster-executor-Upgrade"
	ExecutorScaleOutCondition     TiflowClusterConditionType = "tiflow-cluster-executor-ScaleOut"
	ExecutorScaleInCondition      TiflowClusterConditionType = "tiflow-cluster-executor-ScaleIn"
	ExecutorDecommissionCondition TiflowClusterConditionType = "tiflow-cluster-executor-Decommission"
)
