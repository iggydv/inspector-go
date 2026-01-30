package core_test

import (
	"context"
	"testing"
	"time"

	"inspectgo/pkg/core"
	"inspectgo/pkg/scorer"

	"github.com/stretchr/testify/require"
)

type staticDataset struct {
	samples []core.Sample
}

func (s staticDataset) Name() string {
	return "static"
}

func (s staticDataset) Len(_ context.Context) (int, error) {
	return len(s.samples), nil
}

func (s staticDataset) Samples(ctx context.Context) (<-chan core.Sample, <-chan error) {
	sampleCh := make(chan core.Sample)
	errCh := make(chan error, 1)
	go func() {
		defer close(sampleCh)
		defer close(errCh)
		for _, sample := range s.samples {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case sampleCh <- sample:
			}
		}
	}()
	return sampleCh, errCh
}

type echoSolver struct{}

func (e echoSolver) Name() string {
	return "echo"
}

func (e echoSolver) Solve(_ context.Context, sample core.Sample) (core.Response, error) {
	return core.Response{
		Content: sample.Expected,
		Latency: 5 * time.Millisecond,
	}, nil
}

func TestEvaluatorRun(t *testing.T) {
	ds := staticDataset{
		samples: []core.Sample{
			{ID: "1", Input: "a", Expected: "a"},
			{ID: "2", Input: "b", Expected: "b"},
		},
	}
	eval := core.Evaluator{
		Dataset: ds,
		Solver:  echoSolver{},
		Scorer:  scorer.ExactMatch{CaseSensitive: true, NormalizeWhitespace: true},
		Workers: 2,
	}

	report, err := eval.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, report.Metrics.TotalSamples)
	require.Equal(t, 1.0, report.Metrics.SuccessRate)
}
