package core

import "context"

// Scorer evaluates a model response against a sample.
type Scorer interface {
	Name() string
	Score(ctx context.Context, sample Sample, response Response) (Score, error)
}
