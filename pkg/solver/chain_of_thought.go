package solver

import (
	"context"
	"fmt"

	"inspectgo/pkg/core"
)

// ChainOfThoughtSolver adds a reasoning prefix to the prompt.
type ChainOfThoughtSolver struct {
	Model          core.Model
	Options        core.GenerateOptions
	PromptTemplate string
	ReasoningHint  string
}

func (c ChainOfThoughtSolver) Name() string {
	if c.Model == nil {
		return "chain-of-thought"
	}
	return c.Model.Name()
}

func (c ChainOfThoughtSolver) Solve(ctx context.Context, sample core.Sample) (core.Response, error) {
	if c.Model == nil {
		return core.Response{}, fmt.Errorf("solver: model is required")
	}
	hint := c.ReasoningHint
	if hint == "" {
		hint = "Let's think step by step."
	}

	prompt := sample.Input
	if c.PromptTemplate != "" {
		prompt = applyTemplate(c.PromptTemplate, map[string]string{
			"input": sample.Input,
		})
	}
	prompt = prompt + "\n\n" + hint
	return c.Model.Generate(ctx, prompt, c.Options)
}
