package model

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"inspectgo/pkg/core"
)

const defaultAnthropicModel = "claude-3-5-haiku-latest"

type AnthropicModel struct {
	Client     anthropic.Client
	Model      string
	Timeout    time.Duration
	MaxRetries int
	Backoff    time.Duration
	MaxTokens  int
}

func NewAnthropicModelFromEnv(modelName string) (*AnthropicModel, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, errors.New("anthropic: ANTHROPIC_API_KEY is required")
	}
	if modelName == "" {
		modelName = defaultAnthropicModel
	}
	return &AnthropicModel{
		Client:     anthropic.NewClient(option.WithAPIKey(apiKey)),
		Model:      modelName,
		Timeout:    30 * time.Second,
		MaxRetries: 2,
		Backoff:    500 * time.Millisecond,
		MaxTokens:  1024,
	}, nil
}

func (a AnthropicModel) Name() string {
	if a.Model == "" {
		return defaultAnthropicModel
	}
	return a.Model
}

func (a AnthropicModel) Generate(ctx context.Context, prompt string, opts core.GenerateOptions) (core.Response, error) {
	modelName := a.Name()
	timeout := a.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	maxRetries := a.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	backoff := a.Backoff
	if backoff <= 0 {
		backoff = 500 * time.Millisecond
	}
	maxTokens := a.MaxTokens
	if opts.MaxTokens > 0 {
		maxTokens = opts.MaxTokens
	}
	if maxTokens <= 0 {
		maxTokens = 1024
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(modelName),
		MaxTokens: int64(maxTokens),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	}
	if opts.SystemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: opts.SystemPrompt},
		}
	}
	if opts.Temperature > 0 {
		params.Temperature = anthropic.Float(float64(opts.Temperature))
	}
	if opts.TopP > 0 {
		params.TopP = anthropic.Float(float64(opts.TopP))
	}
	if len(opts.Stop) > 0 {
		params.StopSequences = opts.Stop
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, timeout)
		start := time.Now()
		message, err := a.Client.Messages.New(attemptCtx, params)
		cancel()
		if err == nil {
			content := extractAnthropicText(message.Content)
			usage := core.TokenUsage{}
			usage.PromptTokens = int(message.Usage.InputTokens)
			usage.CompletionTokens = int(message.Usage.OutputTokens)
			usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
			return core.Response{
				Content:    content,
				TokenUsage: usage,
				Latency:    time.Since(start),
			}, nil
		}

		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return core.Response{}, err
		}
		lastErr = err
		if attempt < maxRetries {
			select {
			case <-ctx.Done():
				return core.Response{}, ctx.Err()
			case <-time.After(backoff * time.Duration(attempt+1)):
			}
		}
	}

	return core.Response{}, fmt.Errorf("anthropic: request failed after retries: %w", lastErr)
}

func extractAnthropicText(blocks []anthropic.ContentBlockUnion) string {
	if len(blocks) == 0 {
		return ""
	}
	var builder strings.Builder
	for _, block := range blocks {
		if block.Type == "text" {
			builder.WriteString(block.Text)
		}
	}
	return builder.String()
}
