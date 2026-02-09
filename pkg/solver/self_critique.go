package solver

import (
	"context"
	"fmt"
	"time"

	"inspectgo/pkg/core"
)

// SelfCritiqueSolver runs a 3-step solve: initial answer, critique, and revision.
// When SkipInitial is true, sample.Input is treated as an existing answer and
// only critique + revision are performed (useful after chaining with another solver).
type SelfCritiqueSolver struct {
	Model            core.Model
	Options          core.GenerateOptions
	PromptTemplate   string
	CritiqueTemplate string
	ReviseTemplate   string
	SkipInitial      bool
}

func (s SelfCritiqueSolver) Name() string {
	if s.Model == nil {
		return "self-critique"
	}
	return s.Model.Name()
}

func (s SelfCritiqueSolver) Solve(ctx context.Context, sample core.Sample) (core.Response, error) {
	if s.Model == nil {
		return core.Response{}, fmt.Errorf("solver: model is required")
	}

	opts := s.Options

	var totalUsage core.TokenUsage
	var totalLatency time.Duration

	// Step 1: Initial answer (skipped when SkipInitial is true)
	answer := sample.Input
	if !s.SkipInitial {
		promptTmpl := s.PromptTemplate
		if promptTmpl == "" {
			promptTmpl = "{{input}}"
		}
		initialOpts := opts
		if initialOpts.SystemPrompt == "" {
			initialOpts.SystemPrompt = "Return only the final answer with no extra text."
		}
		if initialOpts.MaxTokens == 0 {
			initialOpts.MaxTokens = 512
		}

		prompt := applyTemplate(promptTmpl, map[string]string{"input": sample.Input})
		resp, err := s.Model.Generate(ctx, prompt, initialOpts)
		if err != nil {
			return core.Response{}, err
		}
		answer = resp.Content
		totalUsage = addTokenUsage(totalUsage, resp.TokenUsage)
		totalLatency += resp.Latency
	}

	// Step 2: Critique
	critiqueTmpl := s.CritiqueTemplate
	if critiqueTmpl == "" {
		critiqueTmpl = "Question: {{input}}\nAnswer: {{answer}}\n\nIdentify any errors in the answer above."
	}
	critiqueOpts := opts
	if critiqueOpts.SystemPrompt == "" {
		critiqueOpts.SystemPrompt = "You are a critical reviewer. Identify errors, logical flaws, or incorrect reasoning. Be concise."
	}
	if critiqueOpts.MaxTokens == 0 {
		critiqueOpts.MaxTokens = 512
	}

	critiquePrompt := applyTemplate(critiqueTmpl, map[string]string{
		"input":  sample.Input,
		"answer": answer,
	})
	critiqueResp, err := s.Model.Generate(ctx, critiquePrompt, critiqueOpts)
	if err != nil {
		return core.Response{}, err
	}
	totalUsage = addTokenUsage(totalUsage, critiqueResp.TokenUsage)
	totalLatency += critiqueResp.Latency

	// Step 3: Revise
	reviseTmpl := s.ReviseTemplate
	if reviseTmpl == "" {
		reviseTmpl = "Question: {{input}}\nAnswer: {{answer}}\nCritique: {{critique}}\n\nProvide a corrected final answer."
	}
	reviseOpts := opts
	if reviseOpts.SystemPrompt == "" {
		reviseOpts.SystemPrompt = "Given the critique, provide a corrected final answer. Return only the answer with no extra text."
	}
	if reviseOpts.MaxTokens == 0 {
		reviseOpts.MaxTokens = 256
	}

	revisePrompt := applyTemplate(reviseTmpl, map[string]string{
		"input":    sample.Input,
		"answer":   answer,
		"critique": critiqueResp.Content,
	})
	reviseResp, err := s.Model.Generate(ctx, revisePrompt, reviseOpts)
	if err != nil {
		return core.Response{}, err
	}
	totalUsage = addTokenUsage(totalUsage, reviseResp.TokenUsage)
	totalLatency += reviseResp.Latency

	return core.Response{
		Content:    reviseResp.Content,
		TokenUsage: totalUsage,
		Latency:    totalLatency,
	}, nil
}

func addTokenUsage(a, b core.TokenUsage) core.TokenUsage {
	return core.TokenUsage{
		PromptTokens:     a.PromptTokens + b.PromptTokens,
		CompletionTokens: a.CompletionTokens + b.CompletionTokens,
		TotalTokens:      a.TotalTokens + b.TotalTokens,
	}
}
