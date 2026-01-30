package solver

import (
	"context"
	"testing"

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
