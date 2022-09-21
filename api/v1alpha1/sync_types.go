package v1alpha1

// SyncTypeName type alias
type SyncTypeName int

// All possible Sync type's name
const (
	CreateType SyncTypeName = iota
	UpdateType
	ScaleOutType
	ScaleInType
	ScaleUpType
	ScaleDownType
	DeleteTypeType
	RunType
	UnknownType
)

var syncNames []string = []string{
	"Create",
	"Update",
	"ScaleOut",
	"ScaleIn",
	"ScaleUp",
	"ScaleDown",
	"Delete",
	"Run",
	"Unknown",
}

func (a SyncTypeName) String() string {
	if a < CreateType || a > RunType {
		return "Unknown"
	}
	return syncNames[a]
}
