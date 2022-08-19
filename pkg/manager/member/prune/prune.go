package prune

import "context"

type PVCPruner interface {
	Prune(ctx context.Context) error
	IsStateful() bool
}
