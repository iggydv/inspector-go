package solver

import (
	"context"
	"fmt"
	"time"

	"inspectgo/pkg/core"
)

// MultiStepSolver runs multiple sequential generations.
type MultiStepSolver struct {
	Model         core.Model
	Options       core.GenerateOptions
	Steps         int
	StepTemplate  string
	FinalTemplate string
}

func (m MultiStepSolver) Name() string {
	if m.Model == nil {
		return "multi-step"
	}
	return m.Model.Name()
}

func (m MultiStepSolver) Solve(ctx context.Context, sample core.Sample) (core.Response, error) {
	if m.Model == nil {
		return core.Response{}, fmt.Errorf("solver: model is required")
	}
	steps := m.Steps
	if steps <= 0 {
		steps = 2
	}

	stepTemplate := m.StepTemplate
	if stepTemplate == "" {
		stepTemplate = "Step {{step}}/{{total}}:\nInput: {{input}}\nPrevious: {{previous}}\nAnswer:"
	}

	var totalUsage core.TokenUsage
	var totalLatency time.Duration
	previous := ""

	for i := 1; i <= steps; i++ {
		prompt := applyTemplate(stepTemplate, map[string]string{
			"step":     fmt.Sprintf("%d", i),
			"total":    fmt.Sprintf("%d", steps),
			"input":    sample.Input,
			"previous": previous,
		})
		response, err := m.Model.Generate(ctx, prompt, m.Options)
		if err != nil {
			return core.Response{}, err
		}
		totalUsage = addTokenUsage(totalUsage, response.TokenUsage)
		totalLatency += response.Latency
		previous = response.Content
	}

	if m.FinalTemplate != "" {
		finalPrompt := applyTemplate(m.FinalTemplate, map[string]string{
			"input":    sample.Input,
			"previous": previous,
		})
		finalResp, err := m.Model.Generate(ctx, finalPrompt, m.Options)
		if err != nil {
			return core.Response{}, err
		}
		totalUsage = addTokenUsage(totalUsage, finalResp.TokenUsage)
		totalLatency += finalResp.Latency
		return core.Response{
			Content:    finalResp.Content,
			TokenUsage: totalUsage,
			Latency:    totalLatency,
		}, nil
	}

	return core.Response{
		Content:    previous,
		TokenUsage: totalUsage,
		Latency:    totalLatency,
	}, nil
}
