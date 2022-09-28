package status

type SyncPhaseManager interface {
	SyncPhase()
}

type UpdateStatus interface {
	Update() error
}
