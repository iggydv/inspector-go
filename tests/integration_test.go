package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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
