package solver

import (
	"context"
	"fmt"
	"testing"
	"time"

	"inspectgo/pkg/core"
	"inspectgo/pkg/model"

	"github.com/stretchr/testify/require"
)

func TestChainOfThoughtSolver(t *testing.T) {
	s := ChainOfThoughtSolver{Model: model.MockModel{}}
	resp, err := s.Solve(context.Background(), core.Sample{Input: "test"})
	require.NoError(t, err)
	require.Contains(t, resp.Content, "test")
}

func TestFewShotSolver(t *testing.T) {
	s := FewShotSolver{
		Model: model.MockModel{},
		Examples: []FewShotExample{
			{Input: "a", Output: "1"},
		},
	}
	resp, err := s.Solve(context.Background(), core.Sample{Input: "b"})
	require.NoError(t, err)
	require.Contains(t, resp.Content, "Q: a")
	require.Contains(t, resp.Content, "Q: b")
}

func TestMultiStepSolver(t *testing.T) {
	s := MultiStepSolver{Model: model.MockModel{}, Steps: 2}
	resp, err := s.Solve(context.Background(), core.Sample{Input: "hello"})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Content)
}

func TestSelfConsistencySolver(t *testing.T) {
	s := SelfConsistencySolver{Model: model.MockModel{}, Samples: 2}
	resp, err := s.Solve(context.Background(), core.Sample{Input: "ping"})
	require.NoError(t, err)
	require.Contains(t, resp.Content, "ping")
}

type echoTool struct{}

func (e echoTool) Name() string {
	return "echo"
}

func (e echoTool) Description() string {
	return "echo tool"
}

func (e echoTool) Call(_ context.Context, input string) (string, error) {
	return "echo:" + input, nil
}

func TestToolUseSolver(t *testing.T) {
	s := ToolUseSolver{
		Model: model.MockModel{},
		Tools: []Tool{echoTool{}},
	}
	resp, err := s.Solve(context.Background(), core.Sample{Input: "hi"})
	require.NoError(t, err)
	require.Contains(t, resp.Content, "echo:hi")
}

// countingMockModel tracks how many times Generate is called and returns
// a configurable sequence of responses.
type countingMockModel struct {
	Responses []string
	CallCount int
}

func (m *countingMockModel) Name() string { return "counting-mock" }

func (m *countingMockModel) Generate(_ context.Context, prompt string, _ core.GenerateOptions) (core.Response, error) {
	idx := m.CallCount
	m.CallCount++
	content := prompt
	if idx < len(m.Responses) {
		content = m.Responses[idx]
	}
	return core.Response{
		Content: content,
		TokenUsage: core.TokenUsage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
		Latency: 10 * time.Millisecond,
	}, nil
}

func TestExtractFinalAnswer(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "the answer is pattern",
			input:    "First I add 2+2=4.\nThe answer is: 4",
			expected: "4",
		},
		{
			name:     "the final answer is pattern",
			input:    "Working through this...\nThe final answer is 42",
			expected: "42",
		},
		{
			name:     "GSM8K #### pattern",
			input:    "Step 1: ...\nStep 2: ...\n#### 123",
			expected: "123",
		},
		{
			name:     "therefore pattern",
			input:    "2+3=5\nTherefore, 5",
			expected: "5",
		},
		{
			name:     "therefore without comma",
			input:    "Calculation: 10\nTherefore 10 is the result",
			expected: "10 is the result",
		},
		{
			name:     "fallback to last line",
			input:    "Some reasoning\nMore reasoning\n42",
			expected: "42",
		},
		{
			name:     "single line",
			input:    "42",
			expected: "42",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "case insensitive",
			input:    "THE ANSWER IS: hello",
			expected: "hello",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractFinalAnswer(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestChainOfThoughtExtraction(t *testing.T) {
	mock := &countingMockModel{
		Responses: []string{
			"Let me think step by step.\n2+2=4\nThe answer is: 4",
		},
	}
	s := ChainOfThoughtSolver{
		Model:         mock,
		ExtractAnswer: true,
	}

	resp, err := s.Solve(context.Background(), core.Sample{Input: "What is 2+2?"})
	require.NoError(t, err)
	require.Equal(t, "4", resp.Content)
	require.Equal(t, 1, mock.CallCount)
}

func TestChainOfThoughtNoExtraction(t *testing.T) {
	mock := &countingMockModel{
		Responses: []string{
			"Let me think step by step.\n2+2=4\nThe answer is: 4",
		},
	}
	s := ChainOfThoughtSolver{
		Model:         mock,
		ExtractAnswer: false,
	}

	resp, err := s.Solve(context.Background(), core.Sample{Input: "What is 2+2?"})
	require.NoError(t, err)
	require.Contains(t, resp.Content, "Let me think")
	require.Contains(t, resp.Content, "The answer is: 4")
}

func TestSelfCritiqueSolver(t *testing.T) {
	mock := &countingMockModel{
		Responses: []string{
			"initial answer: 5",     // Step 1: initial
			"the answer seems off",  // Step 2: critique
			"corrected answer: 42",  // Step 3: revision
		},
	}
	s := SelfCritiqueSolver{Model: mock}

	resp, err := s.Solve(context.Background(), core.Sample{
		Input:    "What is the meaning of life?",
		Expected: "42",
	})
	require.NoError(t, err)
	require.Equal(t, "corrected answer: 42", resp.Content)
	require.Equal(t, 3, mock.CallCount)

	// Verify token aggregation across 3 calls
	require.Equal(t, 30, resp.TokenUsage.PromptTokens)     // 10 * 3
	require.Equal(t, 15, resp.TokenUsage.CompletionTokens)  // 5 * 3
	require.Equal(t, 45, resp.TokenUsage.TotalTokens)       // 15 * 3
	require.Equal(t, 30*time.Millisecond, resp.Latency)     // 10ms * 3
}

func TestSelfCritiqueSolverSkipInitial(t *testing.T) {
	mock := &countingMockModel{
		Responses: []string{
			"the answer has an error", // Step 2: critique (step 1 skipped)
			"corrected: 7",            // Step 3: revision
		},
	}
	s := SelfCritiqueSolver{
		Model:       mock,
		SkipInitial: true,
	}

	resp, err := s.Solve(context.Background(), core.Sample{
		Input:    "previous solver said 6",
		Expected: "7",
	})
	require.NoError(t, err)
	require.Equal(t, "corrected: 7", resp.Content)
	require.Equal(t, 2, mock.CallCount) // Only critique + revision

	// Verify token aggregation across 2 calls (no initial)
	require.Equal(t, 20, resp.TokenUsage.PromptTokens)
	require.Equal(t, 10, resp.TokenUsage.CompletionTokens)
	require.Equal(t, 30, resp.TokenUsage.TotalTokens)
}

func TestPipelineSolver(t *testing.T) {
	mock1 := &countingMockModel{
		Responses: []string{"stage1 output"},
	}
	mock2 := &countingMockModel{
		Responses: []string{"stage2 output"},
	}

	s1 := BasicSolver{Model: mock1, PromptTemplate: "{{input}}"}
	s2 := BasicSolver{Model: mock2, PromptTemplate: "{{input}}"}
	pipeline := PipelineSolver{Solvers: []core.Solver{s1, s2}}

	resp, err := pipeline.Solve(context.Background(), core.Sample{
		ID:       "test-1",
		Input:    "original input",
		Expected: "expected output",
	})
	require.NoError(t, err)
	require.Equal(t, "stage2 output", resp.Content)

	// Verify stage1 was called with original input
	require.Equal(t, 1, mock1.CallCount)
	// Verify stage2 was called (with stage1's output as input)
	require.Equal(t, 1, mock2.CallCount)

	// Verify token aggregation
	require.Equal(t, 20, resp.TokenUsage.PromptTokens)
	require.Equal(t, 10, resp.TokenUsage.CompletionTokens)
	require.Equal(t, 30, resp.TokenUsage.TotalTokens)
}

func TestPipelineSolverName(t *testing.T) {
	s1 := BasicSolver{Model: model.MockModel{NameValue: "model-a"}}
	s2 := ChainOfThoughtSolver{Model: model.MockModel{NameValue: "model-b"}}

	pipeline := PipelineSolver{Solvers: []core.Solver{s1, s2}}
	require.Equal(t, "model-a | model-b", pipeline.Name())
}

func TestPipelineSolverPreservesMetadata(t *testing.T) {
	// Use a mock that echoes the prompt so we can verify what stage2 received
	echoMock := model.MockModel{}
	s1 := BasicSolver{Model: model.MockModel{ResponseText: "stage1-result"}, PromptTemplate: "{{input}}"}
	s2 := BasicSolver{Model: echoMock, PromptTemplate: "{{input}}"}

	pipeline := PipelineSolver{Solvers: []core.Solver{s1, s2}}

	sample := core.Sample{
		ID:       "id-42",
		Input:    "original",
		Expected: "keep-this",
		Metadata: map[string]string{"key": "value"},
	}

	resp, err := pipeline.Solve(context.Background(), sample)
	require.NoError(t, err)
	// Stage2 received stage1's output as input
	require.Equal(t, "stage1-result", resp.Content)
}

func TestMultiStepTokenAggregation(t *testing.T) {
	mock := &countingMockModel{
		Responses: []string{"step1", "step2"},
	}
	s := MultiStepSolver{Model: mock, Steps: 2}

	resp, err := s.Solve(context.Background(), core.Sample{Input: "test"})
	require.NoError(t, err)
	require.Equal(t, "step2", resp.Content)
	require.Equal(t, 2, mock.CallCount)

	// Verify tokens aggregated across both steps
	require.Equal(t, 20, resp.TokenUsage.PromptTokens)
	require.Equal(t, 10, resp.TokenUsage.CompletionTokens)
	require.Equal(t, 30, resp.TokenUsage.TotalTokens)
}

func TestSelfConsistencyParallel(t *testing.T) {
	// Verify that self-consistency with a fixed response works correctly
	mock := model.MockModel{ResponseText: "42"}
	s := SelfConsistencySolver{
		Model:   mock,
		Samples: 5,
	}

	resp, err := s.Solve(context.Background(), core.Sample{Input: "test"})
	require.NoError(t, err)
	require.Equal(t, "42", resp.Content)
}

func TestPipelineSolverEmpty(t *testing.T) {
	pipeline := PipelineSolver{}
	_, err := pipeline.Solve(context.Background(), core.Sample{Input: "test"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one solver")
}

func TestPipelineSolverChainOfThoughtThenSelfCritique(t *testing.T) {
	mock := &countingMockModel{
		Responses: []string{
			"Let me think...\nThe answer is: 5", // CoT response
			"The answer looks correct",           // Critique
			"5",                                   // Revision
		},
	}

	cot := ChainOfThoughtSolver{Model: mock, ExtractAnswer: true}
	critique := SelfCritiqueSolver{Model: mock, SkipInitial: true}
	pipeline := PipelineSolver{Solvers: []core.Solver{cot, critique}}

	resp, err := pipeline.Solve(context.Background(), core.Sample{
		Input:    "What is 2+3?",
		Expected: "5",
	})
	require.NoError(t, err)
	require.Equal(t, "5", resp.Content)
	require.Equal(t, 3, mock.CallCount) // CoT(1) + Critique(1) + Revise(1)

	// Total: 3 calls * 15 tokens each = 45
	require.Equal(t, 45, resp.TokenUsage.TotalTokens)
}

func TestBasicSolverSmartDefaults(t *testing.T) {
	// Verify that BasicSolver passes system prompt and max_tokens to model
	capturer := &optCapturingModel{}
	s := BasicSolver{Model: capturer, PromptTemplate: "{{input}}"}

	_, err := s.Solve(context.Background(), core.Sample{Input: "test"})
	require.NoError(t, err)
	require.Equal(t, "Return only the final answer with no extra text.", capturer.LastOpts.SystemPrompt)
	require.Equal(t, 256, capturer.LastOpts.MaxTokens)
}

func TestChainOfThoughtSmartDefaults(t *testing.T) {
	capturer := &optCapturingModel{}
	s := ChainOfThoughtSolver{Model: capturer}

	_, err := s.Solve(context.Background(), core.Sample{Input: "test"})
	require.NoError(t, err)
	require.Contains(t, capturer.LastOpts.SystemPrompt, "step by step")
	require.Equal(t, 1024, capturer.LastOpts.MaxTokens)
}

func TestSolverRespectsUserOpts(t *testing.T) {
	capturer := &optCapturingModel{}
	userOpts := core.GenerateOptions{
		SystemPrompt: "custom system",
		MaxTokens:    999,
		Temperature:  0.5,
	}

	s := BasicSolver{Model: capturer, Options: userOpts, PromptTemplate: "{{input}}"}
	_, err := s.Solve(context.Background(), core.Sample{Input: "test"})
	require.NoError(t, err)
	// User-provided values should be preserved
	require.Equal(t, "custom system", capturer.LastOpts.SystemPrompt)
	require.Equal(t, 999, capturer.LastOpts.MaxTokens)
	require.Equal(t, float32(0.5), capturer.LastOpts.Temperature)
}

// optCapturingModel captures the GenerateOptions passed to it.
type optCapturingModel struct {
	LastOpts   core.GenerateOptions
	LastPrompt string
}

func (m *optCapturingModel) Name() string { return "opt-capturer" }

func (m *optCapturingModel) Generate(_ context.Context, prompt string, opts core.GenerateOptions) (core.Response, error) {
	m.LastOpts = opts
	m.LastPrompt = prompt
	return core.Response{Content: fmt.Sprintf("response to: %s", prompt)}, nil
}
