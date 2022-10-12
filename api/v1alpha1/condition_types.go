package v1alpha1

// TiflowClusterConditionType type alias
type TiflowClusterConditionType string

const (
	VersionChecked TiflowClusterConditionType = "VersionChecked"
	LeaderChecked  TiflowClusterConditionType = "LeaderChecked"

	MastersInfoUpdatedChecked TiflowClusterConditionType = "MastersInfoUpdatedChecked"
	MasterVersionChecked      TiflowClusterConditionType = "MasterVersionChecked"
	MasterReplicaChecked      TiflowClusterConditionType = "MasterReplicaChecked"
	MasterReadyChecked        TiflowClusterConditionType = "MasterReadyChecked"
	MasterMembersChecked      TiflowClusterConditionType = "MasterMembersChecked"
	MasterSyncChecked         TiflowClusterConditionType = "MasterSyncChecked"

	ExecutorsInfoUpdatedChecked TiflowClusterConditionType = "ExecutorsInfoUpdatedCheck"
	ExecutorVersionChecked      TiflowClusterConditionType = "ExecutorVersionChecked"
	ExecutorReplicaChecked      TiflowClusterConditionType = "ExecutorReplicaChecked"
	ExecutorReadyChecked        TiflowClusterConditionType = "ExecutorReadyChecked"
	ExecutorPVCChecked          TiflowClusterConditionType = "ExecutorPVCChecked"
	ExecutorMembersChecked      TiflowClusterConditionType = "ExecutorMembersChecked"
	ExecutorSyncChecked         TiflowClusterConditionType = "ExecutorSyncChecked"
)
