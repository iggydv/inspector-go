package core

import "context"

// Task ties together dataset, solver, and scorer.
type Task interface {
	Name() string
	Description() string
	Dataset(ctx context.Context) (Dataset, error)
	Solver() Solver
	Scorer() Scorer
}
