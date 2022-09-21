package v1alpha1

// SyncTypeName type alias
type SyncTypeName int

// All possible Sync type's name
const (
	CreateType SyncTypeName = iota
	UpgradeType
	ScaleOutType
	ScaleInType
	ScaleUpType
	ScaleDownType
	DeleteType
)

func (a SyncTypeName) GetMasterClusterPhase() MasterPhaseType {
	if a < CreateType || a > DeleteType {
		return MasterUnknown
	}
	return syncNameToMasterPhase[a]
}

func (a SyncTypeName) GetExecutorClusterPhase() ExecutorPhaseType {
	if a < CreateType || a > DeleteType {
		return ExecutorUnknown
	}
	return syncNameToExecutorPhase[a]
}

var syncNameToMasterPhase = map[SyncTypeName]MasterPhaseType{
	CreateType:    MasterCreating,
	UpgradeType:   MasterUpgrading,
	ScaleOutType:  MasterScalingOut,
	ScaleInType:   MasterScalingIn,
	ScaleUpType:   MasterScalingUp,
	ScaleDownType: MasterScalingDown,
	DeleteType:    MasterDeleting,
}

var syncNameToExecutorPhase = map[SyncTypeName]ExecutorPhaseType{
	CreateType:    ExecutorCreating,
	UpgradeType:   ExecutorUpgrading,
	ScaleOutType:  ExecutorScalingOut,
	ScaleInType:   ExecutorScalingIn,
	ScaleUpType:   ExecutorScalingUp,
	ScaleDownType: ExecutorScalingDown,
	DeleteType:    ExecutorDeleting,
}
