package v1alpha1

// TiflowClusterConditionType type alias
type TiflowClusterConditionType string

const (
	// CertificateGenerated condition used to run the version checker and sync other actions
	CertificateGenerated TiflowClusterConditionType = "CertificateGenerated"
	// VersionChecked condition used to run the version checker and sync other actions
	VersionChecked TiflowClusterConditionType = "VersionChecked"
	// DecommissionCondition string
	DecommissionCondition TiflowClusterConditionType = "Decommission"
	// InitializedCondition string
	InitializedCondition TiflowClusterConditionType = "Initialized"
	// ClusterRestartCondition string
	ClusterRestartCondition TiflowClusterConditionType = "RestartedCluster"
)
