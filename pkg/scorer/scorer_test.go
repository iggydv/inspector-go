package scorer

import (
	"context"
	"testing"

	"inspectgo/pkg/core"

	"github.com/stretchr/testify/require"
)

func TestExactMatch(t *testing.T) {
	sc := ExactMatch{CaseSensitive: false, NormalizeWhitespace: true}
	sample := core.Sample{Expected: "Hello World"}
	resp := core.Response{Content: "  hello   world  "}

	score, err := sc.Score(context.Background(), sample, resp)
	require.NoError(t, err)
	require.True(t, score.Passed)
	require.Equal(t, 1.0, score.Value)
}

func TestIncludes(t *testing.T) {
	sc := Includes{CaseSensitive: false, NormalizeWhitespace: true}
	sample := core.Sample{Expected: "world"}
	resp := core.Response{Content: "Hello World"}

	score, err := sc.Score(context.Background(), sample, resp)
	require.NoError(t, err)
	require.True(t, score.Passed)
}
