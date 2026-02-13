package model

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"inspectgo/pkg/core"
)

const defaultOllamaBaseURL = "http://localhost:11434/v1"
const defaultOllamaModel = "llama2"

type OllamaModel struct {
	Client     openai.Client
	Model      string
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
	Backoff    time.Duration
}

func NewOllamaModel(baseURL, modelName string) (*OllamaModel, error) {
	if baseURL == "" {
		baseURL = defaultOllamaBaseURL
	}
	if modelName == "" {
		modelName = defaultOllamaModel
	}
	opts := []option.RequestOption{
		option.WithBaseURL(baseURL),
		option.WithAPIKey("ollama"),
	}
	return &OllamaModel{
		Client:     openai.NewClient(opts...),
		Model:      modelName,
		BaseURL:    baseURL,
		Timeout:    60 * time.Second,
		MaxRetries: 2,
		Backoff:    500 * time.Millisecond,
	}, nil
}

func (o OllamaModel) Name() string {
	if o.Model == "" {
		return defaultOllamaModel
	}
	return o.Model
}

func (o OllamaModel) Generate(ctx context.Context, prompt string, opts core.GenerateOptions) (core.Response, error) {
	modelName := o.Name()
	timeout := o.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	maxRetries := o.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	backoff := o.Backoff
	if backoff <= 0 {
		backoff = 500 * time.Millisecond
	}

	messages := []openai.ChatCompletionMessageParamUnion{}
	if opts.SystemPrompt != "" {
		messages = append(messages, openai.SystemMessage(opts.SystemPrompt))
	}
	messages = append(messages, openai.UserMessage(prompt))

	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(modelName),
		Messages: messages,
	}
	if opts.Temperature > 0 {
		params.Temperature = openai.Float(float64(opts.Temperature))
	}
	if opts.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(opts.MaxTokens))
	}
	if opts.TopP > 0 {
		params.TopP = openai.Float(float64(opts.TopP))
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, timeout)
		start := time.Now()
		completion, err := o.Client.Chat.Completions.New(attemptCtx, params)
		cancel()
		if err == nil {
			if len(completion.Choices) == 0 {
				return core.Response{}, fmt.Errorf("ollama: empty response")
			}
			content := completion.Choices[0].Message.Content
			usage := core.TokenUsage{
				PromptTokens:     int(completion.Usage.PromptTokens),
				CompletionTokens: int(completion.Usage.CompletionTokens),
				TotalTokens:      int(completion.Usage.TotalTokens),
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

	return core.Response{}, fmt.Errorf("ollama: request failed after retries: %w", lastErr)
}
