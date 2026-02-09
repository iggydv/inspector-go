package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"inspectgo/pkg/core"
	"inspectgo/pkg/dataset"
	"inspectgo/pkg/model"
	"inspectgo/pkg/scorer"
	"inspectgo/pkg/solver"

	"github.com/stretchr/testify/require"
)

func TestEndToEndEvaluation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "samples.jsonl")
	content := `{"id":"1","input":"ping","expected":"ping"}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	ds := dataset.NewFileDataset(path)
	sc := scorer.ExactMatch{CaseSensitive: true, NormalizeWhitespace: true}
	mockModel := model.MockModel{}
	sv := solver.BasicSolver{
		Model:          mockModel,
		PromptTemplate: "{{input}}",
	}

	eval := core.Evaluator{
		Dataset: ds,
		Solver:  sv,
		Scorer:  sc,
		Workers: 1,
	}

	report, err := eval.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, report.Metrics.TotalSamples)
	require.Equal(t, 1.0, report.Metrics.SuccessRate)
}

func TestChainOfThoughtSelfCritiquePipeline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "samples.jsonl")
	lines := `{"id":"1","input":"What is 2+3?","expected":"5"}
{"id":"2","input":"What is 10-7?","expected":"3"}`
	require.NoError(t, os.WriteFile(path, []byte(lines), 0o600))

	ds := dataset.NewFileDataset(path)
	sc := scorer.Includes{CaseSensitive: false, NormalizeWhitespace: true}

	// Mock that returns answers with "The answer is:" format for CoT extraction
	mockModel := model.MockModel{ResponseText: "Let me think...\nThe answer is: 5"}

	cot := solver.ChainOfThoughtSolver{
		Model:         mockModel,
		ExtractAnswer: true,
	}
	critique := solver.SelfCritiqueSolver{
		Model:       mockModel,
		SkipInitial: true,
	}
	pipeline := solver.PipelineSolver{
		Solvers: []core.Solver{cot, critique},
	}

	eval := core.Evaluator{
		Dataset:      ds,
		Solver:       pipeline,
		Scorer:       sc,
		Workers:      2,
		TotalSamples: 2,
	}

	report, err := eval.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, report.Metrics.TotalSamples)
	require.NotEmpty(t, report.Results)

	// Verify the solver name includes the pipeline
	require.Contains(t, pipeline.Name(), "|")
}

func TestSelfConsistencyPipeline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "samples.jsonl")
	content := `{"id":"1","input":"What is 1+1?","expected":"2"}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	ds := dataset.NewFileDataset(path)
	sc := scorer.Includes{CaseSensitive: false, NormalizeWhitespace: true}
	mockModel := model.MockModel{ResponseText: "2"}

	sv := solver.SelfConsistencySolver{
		Model:   mockModel,
		Samples: 3,
	}

	eval := core.Evaluator{
		Dataset: ds,
		Solver:  sv,
		Scorer:  sc,
		Workers: 1,
	}

	report, err := eval.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, report.Metrics.TotalSamples)
	require.Equal(t, 1.0, report.Metrics.SuccessRate)
}

func TestSampleTimeout(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "samples.jsonl")
	content := `{"id":"1","input":"test","expected":"test"}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	ds := dataset.NewFileDataset(path)
	sc := scorer.ExactMatch{CaseSensitive: true, NormalizeWhitespace: true}
	mockModel := model.MockModel{ResponseText: "test"}
	sv := solver.BasicSolver{
		Model:          mockModel,
		PromptTemplate: "{{input}}",
	}

	eval := core.Evaluator{
		Dataset:       ds,
		Solver:        sv,
		Scorer:        sc,
		Workers:       1,
		SampleTimeout: 5 * time.Second,
	}

	report, err := eval.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, report.Metrics.TotalSamples)
	// Should succeed since mock is instant
	require.Equal(t, "", report.Results[0].Error)
}
