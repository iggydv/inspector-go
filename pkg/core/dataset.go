package core

import "context"

// Dataset provides samples for evaluation.
type Dataset interface {
	Name() string
	Len(ctx context.Context) (int, error)
	Samples(ctx context.Context) (<-chan Sample, <-chan error)
}
