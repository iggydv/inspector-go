package solver

import (
	"context"
	"fmt"

	"inspectgo/pkg/core"
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

	responses := make([]core.Response, 0, count)
	for i := 0; i < count; i++ {
		response, err := s.Model.Generate(ctx, prompt, s.Options)
		if err != nil {
			return core.Response{}, err
		}
		responses = append(responses, response)
	}

	best := responses[0]
	counts := map[string]int{}
	for _, response := range responses {
		counts[response.Content]++
		if counts[response.Content] > counts[best.Content] {
			best = response
		}
	}

	return best, nil
}
