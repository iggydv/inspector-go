package model

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	openai "github.com/sashabaranov/go-openai"

	"inspectgo/pkg/core"
)

const defaultOpenAIModel = "gpt-4o-mini"

type OpenAIModel struct {
	Client     *openai.Client
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
		Client:     openai.NewClient(apiKey),
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
	if o.Client == nil {
		return core.Response{}, errors.New("openai: client is required")
	}

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

	req := openai.ChatCompletionRequest{
		Model: modelName,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
	}
	if opts.Temperature > 0 {
		req.Temperature = opts.Temperature
	}
	if opts.MaxTokens > 0 {
		req.MaxTokens = opts.MaxTokens
	}
	if opts.TopP > 0 {
		req.TopP = opts.TopP
	}
	if len(opts.Stop) > 0 {
		req.Stop = opts.Stop
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, timeout)
		start := time.Now()
		resp, err := o.Client.CreateChatCompletion(attemptCtx, req)
		cancel()
		if err == nil {
			if len(resp.Choices) == 0 {
				return core.Response{}, fmt.Errorf("openai: empty response")
			}
			usage := core.TokenUsage{
				PromptTokens:     resp.Usage.PromptTokens,
				CompletionTokens: resp.Usage.CompletionTokens,
				TotalTokens:      resp.Usage.TotalTokens,
			}
			return core.Response{
				Content:    resp.Choices[0].Message.Content,
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
