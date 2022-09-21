package v1alpha1

// TiflowClusterConditionType type alias
type TiflowClusterConditionType string

const (
	// CertificateGenerated condition used to run to generate certificate and sync other actions
	CertificateGenerated TiflowClusterConditionType = "CertificateGenerated"
	// VersionChecked condition used to run the version checker and sync other actions
	VersionChecked TiflowClusterConditionType = "VersionChecked"
	// DecommissionCondition condition used to decommission nodes and sync other actions
	DecommissionCondition TiflowClusterConditionType = "Decommission"
	// InitializedCondition condition used to init nodes or clusters and sync other actions
	InitializedCondition TiflowClusterConditionType = "Initialized"
	// ClusterRestartCondition condition used to restart clusters and sync other actions
	ClusterRestartCondition TiflowClusterConditionType = "RestartedCluster"
)
