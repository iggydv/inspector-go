package core

import "context"

// Model generates responses for prompts.
type Model interface {
	Name() string
	Generate(ctx context.Context, prompt string, opts GenerateOptions) (Response, error)
}
