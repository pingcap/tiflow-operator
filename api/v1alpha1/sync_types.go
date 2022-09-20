package v1alpha1

// SyncType type alias
type SyncType string

// All possible sync types
const (
	CreatType   SyncType = "Create"
	UpdateType  SyncType = "Update"
	ScaleOut    SyncType = "ScaleOut"
	ScaleIn     SyncType = "ScaleIn"
	ScaleUp     SyncType = "ScaleUp"
	ScaleDown   SyncType = "ScaleDown"
	DeleteType  SyncType = "Delete"
	UnknownType SyncType = "Unknown"
	RunType     SyncType = "Run"
)
