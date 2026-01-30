package solver

import (
	"context"
	"fmt"

	"inspectgo/pkg/core"
)

type Tool interface {
	Name() string
	Description() string
	Call(ctx context.Context, input string) (string, error)
}

type ToolSelector func(sample core.Sample, tools []Tool) (Tool, error)

// ToolUseSolver runs a tool call before prompting the model.
type ToolUseSolver struct {
	Model              core.Model
	Options            core.GenerateOptions
	Tools              []Tool
	PromptTemplate     string
	ToolResultTemplate string
	SelectTool         ToolSelector
}

func (t ToolUseSolver) Name() string {
	if t.Model == nil {
		return "tool-use"
	}
	return t.Model.Name()
}

func (t ToolUseSolver) Solve(ctx context.Context, sample core.Sample) (core.Response, error) {
	if t.Model == nil {
		return core.Response{}, fmt.Errorf("solver: model is required")
	}
	if len(t.Tools) == 0 {
		return core.Response{}, fmt.Errorf("solver: at least one tool is required")
	}

	tool, err := t.pickTool(sample)
	if err != nil {
		return core.Response{}, err
	}

	toolResult, err := tool.Call(ctx, sample.Input)
	if err != nil {
		return core.Response{}, err
	}

	prompt := t.buildPrompt(sample, tool, toolResult)
	return t.Model.Generate(ctx, prompt, t.Options)
}

func (t ToolUseSolver) pickTool(sample core.Sample) (Tool, error) {
	if t.SelectTool != nil {
		return t.SelectTool(sample, t.Tools)
	}
	return t.Tools[0], nil
}

func (t ToolUseSolver) buildPrompt(sample core.Sample, tool Tool, toolResult string) string {
	template := t.PromptTemplate
	if template == "" {
		template = "Tool: {{tool}}\nToolResult: {{tool_result}}\nInput: {{input}}\nAnswer:"
	}
	return applyTemplate(template, map[string]string{
		"tool":        tool.Name(),
		"tool_result": toolResult,
		"input":       sample.Input,
	})
}
