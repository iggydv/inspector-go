package solver

import (
	"context"
	"fmt"

	"inspectgo/pkg/core"

	"golang.org/x/sync/errgroup"
)

// SelfConsistencySolver samples multiple responses and picks the majority.
type SelfConsistencySolver struct {
	Model          core.Model
	Options        core.GenerateOptions
	Samples        int
	PromptTemplate string
}

func (s SelfConsistencySolver) Name() string {
	if s.Model == nil {
		return "self-consistency"
	}
	return s.Model.Name()
}

func (s SelfConsistencySolver) Solve(ctx context.Context, sample core.Sample) (core.Response, error) {
	if s.Model == nil {
		return core.Response{}, fmt.Errorf("solver: model is required")
	}
	count := s.Samples
	if count <= 0 {
		count = 3
	}

	prompt := sample.Input
	if s.PromptTemplate != "" {
		prompt = applyTemplate(s.PromptTemplate, map[string]string{
			"input": sample.Input,
		})
	}

	responses := make([]core.Response, count)
	g, gctx := errgroup.WithContext(ctx)
	for i := 0; i < count; i++ {
		i := i
		g.Go(func() error {
			resp, err := s.Model.Generate(gctx, prompt, s.Options)
			if err != nil {
				return err
			}
			responses[i] = resp
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return core.Response{}, err
	}

	best := responses[0]
	counts := map[string]int{}
	for _, response := range responses {
		counts[response.Content]++
		if counts[response.Content] > counts[best.Content] {
			best = response
		}
	}

	// Aggregate token usage across all samples
	var totalUsage core.TokenUsage
	for _, resp := range responses {
		totalUsage = addTokenUsage(totalUsage, resp.TokenUsage)
	}
	best.TokenUsage = totalUsage

	return best, nil
}
