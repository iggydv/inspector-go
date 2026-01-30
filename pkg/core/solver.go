package core

import "context"

// Solver turns samples into model responses.
type Solver interface {
	Name() string
	Solve(ctx context.Context, sample Sample) (Response, error)
}
