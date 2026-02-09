package core

import "time"

// Response is a model response plus basic telemetry.
type Response struct {
	Content    string        `json:"content" yaml:"content"`
	TokenUsage TokenUsage    `json:"token_usage" yaml:"token_usage"`
	Latency    time.Duration `json:"latency" yaml:"latency"`
}

// TokenUsage captures token accounting for a request.
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens" yaml:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens" yaml:"completion_tokens"`
	TotalTokens      int `json:"total_tokens" yaml:"total_tokens"`
}

// Score represents a numeric score and pass/fail status.
type Score struct {
	Value   float64 `json:"value" yaml:"value"`
	Max     float64 `json:"max" yaml:"max"`
	Passed  bool    `json:"passed" yaml:"passed"`
	Details string  `json:"details,omitempty" yaml:"details,omitempty"`
}

// EvalResult captures the outcome for one sample.
type EvalResult struct {
	Sample   Sample        `json:"sample" yaml:"sample"`
	Response Response      `json:"response" yaml:"response"`
	Score    Score         `json:"score" yaml:"score"`
	Error    string        `json:"error,omitempty" yaml:"error,omitempty"`
	Duration time.Duration `json:"duration" yaml:"duration"`
}

// EvalReport summarizes an evaluation run.
type EvalReport struct {
	TaskName   string            `json:"task_name" yaml:"task_name"`
	ModelName  string            `json:"model_name" yaml:"model_name"`
	ScorerName string            `json:"scorer_name" yaml:"scorer_name"`
	Metrics    Metrics           `json:"metrics" yaml:"metrics"`
	Results    []EvalResult      `json:"results" yaml:"results"`
	Metadata   map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	StartedAt  time.Time         `json:"started_at" yaml:"started_at"`
	FinishedAt time.Time         `json:"finished_at" yaml:"finished_at"`
}

// Metrics aggregates evaluation statistics.
type Metrics struct {
	TotalSamples int           `json:"total_samples" yaml:"total_samples"`
	SuccessRate  float64       `json:"success_rate" yaml:"success_rate"`
	AverageScore float64       `json:"average_score" yaml:"average_score"`
	MedianScore  float64       `json:"median_score" yaml:"median_score"`
	P50Score     float64       `json:"p50_score" yaml:"p50_score"`
	P95Score     float64       `json:"p95_score" yaml:"p95_score"`
	P99Score     float64       `json:"p99_score" yaml:"p99_score"`
	TokenUsage   TokenUsage    `json:"token_usage" yaml:"token_usage"`
	AvgLatency   time.Duration `json:"avg_latency" yaml:"avg_latency"`
	P50Latency   time.Duration `json:"p50_latency" yaml:"p50_latency"`
	P95Latency   time.Duration `json:"p95_latency" yaml:"p95_latency"`
	P99Latency   time.Duration `json:"p99_latency" yaml:"p99_latency"`
}

// GenerateOptions controls model generation behavior.
type GenerateOptions struct {
	Temperature  float32  `json:"temperature" yaml:"temperature"`
	MaxTokens    int      `json:"max_tokens" yaml:"max_tokens"`
	TopP         float32  `json:"top_p" yaml:"top_p"`
	Stop         []string `json:"stop" yaml:"stop"`
	SystemPrompt string   `json:"system_prompt,omitempty" yaml:"system_prompt,omitempty"`
}
