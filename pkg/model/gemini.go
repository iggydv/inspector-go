package model

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"google.golang.org/genai"

	"inspectgo/pkg/core"
)

const defaultGeminiModel = "gemini-2.0-flash"

type GeminiModel struct {
	Client     *genai.Client
	Model      string
	Timeout    time.Duration
	MaxRetries int
	Backoff    time.Duration
}

func NewGeminiModelFromEnv(modelName string) (*GeminiModel, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}
	if apiKey == "" {
		return nil, errors.New("gemini: GEMINI_API_KEY or GOOGLE_API_KEY is required")
	}
	if modelName == "" {
		modelName = defaultGeminiModel
	}
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}
	return &GeminiModel{
		Client:     client,
		Model:      modelName,
		Timeout:    60 * time.Second,
		MaxRetries: 2,
		Backoff:    500 * time.Millisecond,
	}, nil
}

func (g GeminiModel) Name() string {
	if g.Model == "" {
		return defaultGeminiModel
	}
	return g.Model
}

func (g *GeminiModel) Generate(ctx context.Context, prompt string, opts core.GenerateOptions) (core.Response, error) {
	modelName := g.Name()
	timeout := g.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	maxRetries := g.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	backoff := g.Backoff
	if backoff <= 0 {
		backoff = 500 * time.Millisecond
	}

	config := &genai.GenerateContentConfig{}
	if opts.SystemPrompt != "" {
		parts := genai.Text(opts.SystemPrompt)
		if len(parts) > 0 && parts[0] != nil {
			config.SystemInstruction = parts[0]
		}
	}
	if opts.Temperature > 0 {
		config.Temperature = ptrFloat32(float32(opts.Temperature))
	}
	if opts.MaxTokens > 0 {
		config.MaxOutputTokens = int32(opts.MaxTokens)
	}
	if opts.TopP > 0 {
		config.TopP = ptrFloat32(float32(opts.TopP))
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, timeout)
		start := time.Now()
		result, err := g.Client.Models.GenerateContent(attemptCtx, modelName, genai.Text(prompt), config)
		cancel()
		if err == nil {
			content := result.Text()
			if content == "" {
				return core.Response{}, fmt.Errorf("gemini: empty response")
			}
			usage := core.TokenUsage{}
			if result.UsageMetadata != nil {
				usage.PromptTokens = int(result.UsageMetadata.PromptTokenCount)
				usage.CompletionTokens = int(result.UsageMetadata.CandidatesTokenCount)
				usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
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

	return core.Response{}, fmt.Errorf("gemini: request failed after retries: %w", lastErr)
}

func ptrFloat32(x float32) *float32 { return &x }
