package result

type NotReadyErr struct {
	Err error
}

func (e NotReadyErr) Error() string {
	return e.Err.Error()
}

type SyncStatusErr struct {
	Err error
}

func (e SyncStatusErr) Error() string {
	return e.Err.Error()
}

type SyncPhaseErr struct {
	Err error
}

func (e SyncPhaseErr) Error() string {
	return e.Err.Error()
}

type UpdateStatusErr struct {
	Err error
}

func (e UpdateStatusErr) Error() string {
	return e.Err.Error()
}
