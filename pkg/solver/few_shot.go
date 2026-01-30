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

	fullPrompt := strings.Join(parts, separator)
	return f.Model.Generate(ctx, fullPrompt, f.Options)
}
