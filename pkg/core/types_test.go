package core

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEvalReportJSONRoundTrip(t *testing.T) {
	report := EvalReport{
		TaskName:   "task",
		ModelName:  "model",
		ScorerName: "scorer",
		Metrics: Metrics{
			TotalSamples: 2,
			SuccessRate:  0.5,
			AverageScore: 0.5,
		},
		Results: []EvalResult{
			{
				Sample: Sample{
					ID:       "1",
					Input:    "hi",
					Expected: "hi",
				},
				Response: Response{
					Content: "hi",
					TokenUsage: TokenUsage{
						PromptTokens:     1,
						CompletionTokens: 1,
						TotalTokens:      2,
					},
					Latency: 10 * time.Millisecond,
				},
				Score: Score{
					Value:  1,
					Max:    1,
					Passed: true,
				},
			},
		},
	}

	data, err := json.Marshal(report)
	require.NoError(t, err)

	var decoded EvalReport
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, report.TaskName, decoded.TaskName)
	require.Equal(t, report.ModelName, decoded.ModelName)
	require.Equal(t, report.ScorerName, decoded.ScorerName)
	require.Equal(t, report.Metrics.TotalSamples, decoded.Metrics.TotalSamples)
	require.Len(t, decoded.Results, 1)
	require.Equal(t, report.Results[0].Sample.Input, decoded.Results[0].Sample.Input)
}
