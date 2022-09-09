package v1alpha1

// ActionStatus indicates the current status of Reconcile
type ActionStatus int

const (
	// Failed status means that to reconcile of tiflowcluster is failing
	Failed ActionStatus = iota
	// Starting status means that to reconcile of tiflowcluster is starting
	Starting
	// Ongoing status means that to reconcile of tiflowcluster is ongoing
	Ongoing
	// Completed status means that to reconcile of tiflowcluster is completed
	Completed
	// Unknown status means that to reconcile of tiflowcluster is unknown
	Unknown
)

var statuses []string = []string{
	"Failed",
	"Starting",
	"Ongoing",
	"Completed",
	"Unknown",
}

func (a ActionStatus) String() string {
	if a < Failed || a > Unknown {
		return "Unknown"
	}
	return statuses[a]
}
