package v1alpha1

// TiflowClusterPhaseType type alias
type TiflowClusterPhaseType string

const (
	// ClusterStarting indicates the state of operator is starting
	ClusterStarting TiflowClusterPhaseType = "Starting"
	// ClusterReconciling indicates the state of operator is reconciling
	ClusterReconciling TiflowClusterPhaseType = "Reconciling"
	// ClusterCompleted indicates the state of operator is completed
	ClusterCompleted TiflowClusterPhaseType = "Completed"
	// ClusterUnknown indicates the state of operator is unknown
	ClusterUnknown TiflowClusterPhaseType = "Unknown"
	// ClusterFailed indicates the state of operator is failed
	ClusterFailed TiflowClusterPhaseType = "Failed"
)

// MasterPhaseType indicates the cluster's state of masters
type MasterPhaseType string

const (
	MasterRunning     MasterPhaseType = "Running"
	MasterStarting    MasterPhaseType = "Starting"
	MasterCreating    MasterPhaseType = "Creating"
	MasterUpgrading   MasterPhaseType = "Upgrading"
	MasterScalingOut  MasterPhaseType = "ScalingOut"
	MasterScalingIn   MasterPhaseType = "ScalingIn"
	MasterScalingUp   MasterPhaseType = "ScalingUp"
	MasterScalingDown MasterPhaseType = "ScalingDown"
	MasterDeleting    MasterPhaseType = "Deleting"
	MasterFailed      MasterPhaseType = "Failed"
	MasterUnknown     MasterPhaseType = "Unknown"
)

// ExecutorPhaseType indicates the cluster's state of executors
type ExecutorPhaseType string

const (
	ExecutorRunning     ExecutorPhaseType = "Running"
	ExecutorStarting    ExecutorPhaseType = "Starting"
	ExecutorCreating    ExecutorPhaseType = "Creating"
	ExecutorUpgrading   ExecutorPhaseType = "Upgrading"
	ExecutorScalingOut  ExecutorPhaseType = "ScalingOut"
	ExecutorScalingIn   ExecutorPhaseType = "ScalingIn"
	ExecutorScalingUp   ExecutorPhaseType = "ScalingUp"
	ExecutorScalingDown ExecutorPhaseType = "ScalingDown"
	ExecutorDeleting    ExecutorPhaseType = "Deleting"
	ExecutorFailed      ExecutorPhaseType = "Failed"
	ExecutorUnknown     ExecutorPhaseType = "Unknown"
)
