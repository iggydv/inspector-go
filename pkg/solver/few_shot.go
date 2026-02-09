package solver

import (
	"context"
	"fmt"
	"strings"

	"inspectgo/pkg/core"
)

type FewShotExample struct {
	Input  string
	Output string
}

// FewShotSolver prepends example pairs before the prompt.
type FewShotSolver struct {
	Model           core.Model
	Options         core.GenerateOptions
	Examples        []FewShotExample
	PromptTemplate  string
	ExampleTemplate string
	Separator       string
}

func (f FewShotSolver) Name() string {
	if f.Model == nil {
		return "few-shot"
	}
	return f.Model.Name()
}

func (f FewShotSolver) Solve(ctx context.Context, sample core.Sample) (core.Response, error) {
	if f.Model == nil {
		return core.Response{}, fmt.Errorf("solver: model is required")
	}

	separator := f.Separator
	if separator == "" {
		separator = "\n\n"
	}

	exampleTemplate := f.ExampleTemplate
	if exampleTemplate == "" {
		exampleTemplate = "Q: {{input}}\nA: {{output}}"
	}

	var parts []string
	for _, ex := range f.Examples {
		parts = append(parts, applyTemplate(exampleTemplate, map[string]string{
			"input":  ex.Input,
			"output": ex.Output,
		}))
	}

	prompt := sample.Input
	if f.PromptTemplate != "" {
		prompt = applyTemplate(f.PromptTemplate, map[string]string{
			"input": sample.Input,
		})
	} else {
		prompt = "Q: " + sample.Input + "\nA:"
	}
	parts = append(parts, prompt)

	opts := f.Options
	if opts.SystemPrompt == "" {
		opts.SystemPrompt = "Answer the question following the pattern shown in the examples. Return only the final answer."
	}
	if opts.MaxTokens == 0 {
		opts.MaxTokens = 256
	}

	fullPrompt := strings.Join(parts, separator)
	return f.Model.Generate(ctx, fullPrompt, opts)
}
