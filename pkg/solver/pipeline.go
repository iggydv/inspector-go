package solver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"inspectgo/pkg/core"
)

// PipelineSolver composes multiple solvers sequentially. Between stages,
// the previous response content becomes the next sample's Input while
// preserving Expected, ID, and Metadata.
type PipelineSolver struct {
	Solvers []core.Solver
}

func (p PipelineSolver) Name() string {
	if len(p.Solvers) == 0 {
		return "pipeline"
	}
	names := make([]string, len(p.Solvers))
	for i, s := range p.Solvers {
		names[i] = s.Name()
	}
	return strings.Join(names, " | ")
}

func (p PipelineSolver) Solve(ctx context.Context, sample core.Sample) (core.Response, error) {
	if len(p.Solvers) == 0 {
		return core.Response{}, fmt.Errorf("pipeline: at least one solver is required")
	}

	var totalUsage core.TokenUsage
	var totalLatency time.Duration
	current := sample

	var lastResponse core.Response
	for _, s := range p.Solvers {
		resp, err := s.Solve(ctx, current)
		if err != nil {
			return core.Response{}, err
		}
		totalUsage = addTokenUsage(totalUsage, resp.TokenUsage)
		totalLatency += resp.Latency
		lastResponse = resp

		// Feed output as input to next stage
		current = core.Sample{
			ID:       sample.ID,
			Input:    resp.Content,
			Expected: sample.Expected,
			Metadata: sample.Metadata,
		}
	}

	return core.Response{
		Content:    lastResponse.Content,
		TokenUsage: totalUsage,
		Latency:    totalLatency,
	}, nil
}
