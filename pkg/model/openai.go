package model

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"

	"inspectgo/pkg/core"
)

const defaultOpenAIModel = "gpt-4o-mini"

type OpenAIModel struct {
	Client     openai.Client
	Model      string
	Timeout    time.Duration
	MaxRetries int
	Backoff    time.Duration
}

func NewOpenAIModelFromEnv(modelName string) (*OpenAIModel, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("openai: OPENAI_API_KEY is required")
	}
	if modelName == "" {
		modelName = defaultOpenAIModel
	}
	return &OpenAIModel{
		Client:     openai.NewClient(option.WithAPIKey(apiKey)),
		Model:      modelName,
		Timeout:    30 * time.Second,
		MaxRetries: 2,
		Backoff:    500 * time.Millisecond,
	}, nil
}

func (o OpenAIModel) Name() string {
	if o.Model == "" {
		return defaultOpenAIModel
	}
	return o.Model
}

func (o OpenAIModel) Generate(ctx context.Context, prompt string, opts core.GenerateOptions) (core.Response, error) {
	modelName := o.Name()
	timeout := o.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	maxRetries := o.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	backoff := o.Backoff
	if backoff <= 0 {
		backoff = 500 * time.Millisecond
	}

	params := responses.ResponseNewParams{
		Model: openai.ChatModel(modelName),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(prompt),
		},
		Store: openai.Bool(false),
	}
	if opts.SystemPrompt != "" {
		params.Instructions = openai.String(opts.SystemPrompt)
	}
	if opts.Temperature > 0 {
		params.Temperature = openai.Float(float64(opts.Temperature))
	}
	if opts.MaxTokens > 0 {
		params.MaxOutputTokens = openai.Int(int64(opts.MaxTokens))
	}
	if opts.TopP > 0 {
		params.TopP = openai.Float(float64(opts.TopP))
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, timeout)
		start := time.Now()
		resp, err := o.Client.Responses.New(attemptCtx, params)
		cancel()
		if err == nil {
			content := resp.OutputText()
			if content == "" {
				return core.Response{}, fmt.Errorf("openai: empty response")
			}
			usage := core.TokenUsage{
				PromptTokens:     int(resp.Usage.InputTokens),
				CompletionTokens: int(resp.Usage.OutputTokens),
				TotalTokens:      int(resp.Usage.TotalTokens),
			}
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

	return core.Response{}, fmt.Errorf("openai: request failed after retries: %w", lastErr)
}
