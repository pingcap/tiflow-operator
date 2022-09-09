package result

type NotReadyErr struct {
	Err error
}

func (e NotReadyErr) Error() string {
	return e.Err.Error()
}

type NormalErr struct {
	Err error
}

func (e NormalErr) Error() string {
	return e.Err.Error()
}

type ValidationErr struct {
	Err error
}

func (e ValidationErr) Error() string {
	return e.Err.Error()
}

type SyncStatusErr struct {
	Err error
}

func (e SyncStatusErr) Error() string {
	return e.Err.Error()
}
