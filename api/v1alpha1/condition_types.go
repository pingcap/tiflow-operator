package v1alpha1

// TiflowClusterConditionType type alias
type TiflowClusterConditionType string

const (
	VersionChecked TiflowClusterConditionType = "VersionChecked"
	SyncChecked    TiflowClusterConditionType = "SyncChecked"
	LeaderChecked  TiflowClusterConditionType = "LeaderChecked"

	MastersInfoUpdatedChecked TiflowClusterConditionType = "MastersInfoUpdatedChecked"
	MasterVersionChecked      TiflowClusterConditionType = "MasterVersionChecked"
	MasterNumChecked          TiflowClusterConditionType = "MasterNumChecked"
	MasterReadyChecked        TiflowClusterConditionType = "MasterReadyChecked"
	MasterMembersChecked      TiflowClusterConditionType = "MasterMembersChecked"

	ExecutorsInfoUpdatedChecked TiflowClusterConditionType = "ExecutorsInfoUpdatedCheck"
	ExecutorVersionChecked      TiflowClusterConditionType = "ExecutorVersionChecked"
	ExecutorNumChecked          TiflowClusterConditionType = "ExecutorNumChecked"
	ExecutorReadyChecked        TiflowClusterConditionType = "ExecutorReadyChecked"
	ExecutorPVCChecked          TiflowClusterConditionType = "ExecutorPVCChecked"
	ExecutorMembersChecked      TiflowClusterConditionType = "ExecutorMembersChecked"
)
