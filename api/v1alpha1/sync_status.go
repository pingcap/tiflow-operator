package v1alpha1

// SyncTypeStatus indicates the current status of SyncType
type SyncTypeStatus int

const (
	// Failed status means this SyncType is failed
	Failed SyncTypeStatus = iota
	// Ongoing status means this SyncType is ongoing
	Ongoing
	// Completed status means this SyncType is completed
	Completed
	// Unknown status means this SyncType is unknown
	Unknown
)

var syncStatuses []string = []string{
	"Failed",
	"Ongoing",
	"Completed",
	"Unknown",
}

func (a SyncTypeStatus) String() string {
	if a < Failed || a > Unknown {
		return "Unknown"
	}
	return syncStatuses[a]
}
