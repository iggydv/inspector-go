package solver

import (
	"context"
	"fmt"
	"strings"

	"inspectgo/pkg/core"
)

// BasicSolver builds a prompt from the sample input.
type BasicSolver struct {
	Model          core.Model
	Options        core.GenerateOptions
	PromptTemplate string
}

func (b BasicSolver) Name() string {
	if b.Model == nil {
		return "basic"
	}
	return b.Model.Name()
}

func (b BasicSolver) Solve(ctx context.Context, sample core.Sample) (core.Response, error) {
	if b.Model == nil {
		return core.Response{}, fmt.Errorf("solver: model is required")
	}
	prompt := sample.Input
	if b.PromptTemplate != "" {
		prompt = strings.ReplaceAll(b.PromptTemplate, "{{input}}", sample.Input)
	} else {
		prompt = applyTemplate("Solve the problem and return only the final answer with no extra text.\nInput: {{input}}\nAnswer:", map[string]string{
			"input": sample.Input,
		})
	}
	return b.Model.Generate(ctx, prompt, b.Options)
}
