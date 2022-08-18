package scale

import "context"

type PVCPruner interface {
	Prune(ctx context.Context) error
	IsStateful() bool
}

type PVCMounter interface {
	Mounter(ctx context.Context) error
	IsStateful() bool
}

type ResetAndCheck interface {
	// WaitUntilRunning blocks until the target statefulSet has the expected number of pods running but not necessarily ready
	WaitUntilRunning(ctx context.Context) error
	// WaitUntilHealthy blocks until the target stateful set has exactly `scale` healthy replicas.
	WaitUntilHealthy(ctx context.Context, scale uint) error
}
