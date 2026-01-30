package inspectlog

import (
	"archive/zip"
	"os"
	"testing"
	"time"

	"inspectgo/pkg/core"

	"github.com/stretchr/testify/require"
)

func TestWriteJSON(t *testing.T) {
	report := core.EvalReport{
		TaskName:   "task",
		ModelName:  "mock",
		ScorerName: "exact",
		StartedAt:  time.Now(),
		Metrics: core.Metrics{
			TotalSamples: 1,
			SuccessRate:  1,
			AverageScore: 1,
			MedianScore:  1,
		},
		Results: []core.EvalResult{
			{
				Sample: core.Sample{
					ID:       "1",
					Input:    "hi",
					Expected: "hi",
				},
				Response: core.Response{Content: "hi"},
				Score:    core.Score{Value: 1, Passed: true},
			},
		},
	}

	log := FromReport(report)
	dir := t.TempDir()
	path, err := WriteJSON(dir, log)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(data), `"status": "success"`)
}

func TestWriteEval(t *testing.T) {
	report := core.EvalReport{
		TaskName:   "task",
		ModelName:  "mock",
		ScorerName: "exact",
		StartedAt:  time.Now(),
		Metrics: core.Metrics{
			TotalSamples: 1,
			SuccessRate:  1,
			AverageScore: 1,
			MedianScore:  1,
		},
		Results: []core.EvalResult{
			{
				Sample: core.Sample{
					ID:       "1",
					Input:    "hi",
					Expected: "hi",
				},
				Response: core.Response{Content: "hi"},
				Score:    core.Score{Value: 1, Passed: true},
			},
		},
	}

	log := FromReport(report)
	dir := t.TempDir()
	path, err := WriteEval(dir, log)
	require.NoError(t, err)

	file, err := os.Open(path)
	require.NoError(t, err)
	defer file.Close()

	reader, err := zip.NewReader(file, fileStatSize(t, file))
	require.NoError(t, err)

	paths := make(map[string]bool)
	for _, f := range reader.File {
		paths[f.Name] = true
	}
	require.True(t, paths["_journal/start.json"])
	require.True(t, paths["header.json"])
	require.True(t, paths["summaries.json"])
	require.True(t, paths["samples/1_epoch_1.json"])
}

func fileStatSize(t *testing.T, file *os.File) int64 {
	stat, err := file.Stat()
	require.NoError(t, err)
	return stat.Size()
}
