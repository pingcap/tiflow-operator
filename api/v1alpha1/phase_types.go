package v1alpha1

type TiflowClusterConditionType string

const (
	ClusterStarting    TiflowClusterConditionType = "Starting"
	ClusterReconciling TiflowClusterConditionType = "Reconciling"
	ClusterCompleted   TiflowClusterConditionType = "Completed"
	ClusterPending     TiflowClusterConditionType = "Pending"
	ClusterUnknown     TiflowClusterConditionType = "Unknown"
	ClusterFailed      TiflowClusterConditionType = "Failed"
)

type MasterConditionType string

const (
	MasterRunning       MasterConditionType = "Running"
	MasterCreating      MasterConditionType = "Creating"
	MasterUpgrading     MasterConditionType = "Upgrading"
	MasterPending       MasterConditionType = "Pending"
	MasterScalingOut    MasterConditionType = "ScalingOut"
	MasterScalingIn     MasterConditionType = "ScalingIn"
	MasterScalingUp     MasterConditionType = "ScalingUp"
	MasterScalingDown   MasterConditionType = "ScalingDown"
	MasterDeleting      MasterConditionType = "Deleting"
	MasterDeletePending MasterConditionType = "DeletePending"
)

type ExecutorConditionType string

const (
	ExecutorRunning       ExecutorConditionType = "Running"
	ExecutorCreating      ExecutorConditionType = "Creating"
	ExecutorUpgrade       ExecutorConditionType = "Upgrading"
	ExecutorPending       ExecutorConditionType = "Pending"
	ExecutorScalingOut    ExecutorConditionType = "ScalingOut"
	ExecutorScalingIn     ExecutorConditionType = "ScalingIn"
	ExecutorScalingUp     ExecutorConditionType = "ScalingUp"
	ExecutorScalingDown   ExecutorConditionType = "ScalingDown"
	ExecutorDeleting      ExecutorConditionType = "Deleting"
	ExecutorDeletePending ExecutorConditionType = "DeletePending"
)
