package v1alpha1

// TiflowClusterConditionType type alias
type TiflowClusterConditionType string

const (
	// VersionChecked condition used to run the version checker and sync other actions
	VersionChecked TiflowClusterConditionType = "VersionChecked"
	// DecommissionCondition condition used to decommission nodes and sync other actions
	DecommissionCondition TiflowClusterConditionType = "Decommission"
	// InitializedCondition condition used to init nodes or clusters and sync other actions
	InitializedCondition TiflowClusterConditionType = "Initialized"

	MastersInfoUpdatedChecked TiflowClusterConditionType = "MastersInfoUpdatedChecked"
	MasterVersionChecked      TiflowClusterConditionType = "MasterVersionChecked"
	MasterNumChecked          TiflowClusterConditionType = "MasterNumChecked"
	MasterReadyChecked        TiflowClusterConditionType = "MasterReadyChecked"
	MasterSynced              TiflowClusterConditionType = "MasterSynced"

	ExecutorsInfoUpdatedChecked TiflowClusterConditionType = "ExecutorsInfoUpdatedCheck"
	ExecutorVersionChecked      TiflowClusterConditionType = "ExecutorVersionChecked"
	ExecutorNumChecked          TiflowClusterConditionType = "ExecutorNumChecked"
	ExecutorReadyChecked        TiflowClusterConditionType = "ExecutorReadyChecked"
	ExecutorSynced              TiflowClusterConditionType = "ExecutorSynced"
	ExecutorPVCChecked          TiflowClusterConditionType = "ExecutorPVCChecked"
)
