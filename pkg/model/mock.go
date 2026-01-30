package model

import (
	"context"
	"time"

	"inspectgo/pkg/core"
)

// MockModel returns a fixed response or echoes the prompt.
type MockModel struct {
	NameValue    string
	ResponseText string
}

func (m MockModel) Name() string {
	if m.NameValue == "" {
		return "mock"
	}
	return m.NameValue
}

func (m MockModel) Generate(_ context.Context, prompt string, _ core.GenerateOptions) (core.Response, error) {
	start := time.Now()
	content := prompt
	if m.ResponseText != "" {
		content = m.ResponseText
	}
	return core.Response{
		Content: content,
		Latency: time.Since(start),
	}, nil
}
