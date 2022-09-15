package v1alpha1

// ActionType type alias
type ActionType string

// All possible action types
const (
	VersionCheckerAction    ActionType = "VersionCheckerAction"
	ClusterRestartAction    ActionType = "ClusterRestart"
	DeployAction            ActionType = "Deploy"
	DecommissionAction      ActionType = "Decommission"
	InitializeAction        ActionType = "Initialize"
	GenerateCertAction      ActionType = "GenerateCert"
	ResizePVCAction         ActionType = "ResizePVC"
	PartitionedUpdateAction ActionType = "PartitionedUpdate"
	SetupRBACAction         ActionType = "SetupRBAC"
	UnknownAction           ActionType = "Unknown"
	ExposeIngressAction     ActionType = "ExposeIngressAction"
)
