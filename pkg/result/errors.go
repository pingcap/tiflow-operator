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

type SyncConditionErr struct {
	Err error
}

func (e SyncConditionErr) Error() string {
	return e.Err.Error()
}

type SyncPhaseErr struct {
	Err error
}

func (e SyncPhaseErr) Error() string {
	return e.Err.Error()
}
