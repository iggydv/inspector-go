package solver

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"inspectgo/pkg/core"
)

// ChainOfThoughtSolver adds a reasoning prefix to the prompt.
type ChainOfThoughtSolver struct {
	Model          core.Model
	Options        core.GenerateOptions
	PromptTemplate string
	ReasoningHint  string
	ExtractAnswer  bool
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

	opts := c.Options
	if opts.SystemPrompt == "" {
		opts.SystemPrompt = "Think step by step. End your response with 'The answer is: <answer>'"
	}
	if opts.MaxTokens == 0 {
		opts.MaxTokens = 1024
	}

	prompt := sample.Input
	if c.PromptTemplate != "" {
		prompt = applyTemplate(c.PromptTemplate, map[string]string{
			"input": sample.Input,
		})
	}
	prompt = prompt + "\n\n" + hint

	response, err := c.Model.Generate(ctx, prompt, opts)
	if err != nil {
		return core.Response{}, err
	}

	if c.ExtractAnswer {
		response.Content = ExtractFinalAnswer(response.Content)
	}

	return response, nil
}

var answerPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)the (?:final )?answer is[:\s]+(.+)`),
	regexp.MustCompile(`####\s*(.+)`),
	regexp.MustCompile(`(?i)therefore,?\s+(.+)`),
}

// ExtractFinalAnswer extracts a clean final answer from model reasoning output.
func ExtractFinalAnswer(text string) string {
	for _, pattern := range answerPatterns {
		matches := pattern.FindStringSubmatch(text)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}
	// Fallback: last non-empty line
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			return line
		}
	}
	return strings.TrimSpace(text)
}
