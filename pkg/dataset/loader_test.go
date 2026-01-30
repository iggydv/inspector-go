package dataset

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"inspectgo/pkg/core"

	"github.com/stretchr/testify/require"
)

func TestFileDatasetJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "samples.json")

	samples := []core.Sample{
		{ID: "1", Input: "a", Expected: "a"},
		{ID: "2", Input: "b", Expected: "b"},
	}
	data, err := json.Marshal(samples)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o600))

	ds := NewFileDataset(path)
	count, err := ds.Len(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, count)

	ch, errCh := ds.Samples(context.Background())
	var got []core.Sample
	for sample := range ch {
		got = append(got, sample)
	}
	for err := range errCh {
		require.NoError(t, err)
	}
	require.Len(t, got, 2)
	require.Equal(t, "a", got[0].Input)
}

func TestFileDatasetJSONL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "samples.jsonl")

	lines := []string{
		`{"id":"1","input":"x","expected":"x"}`,
		`{"id":"2","input":"y","expected":"y"}`,
	}
	require.NoError(t, os.WriteFile(path, []byte(lines[0]+"\n"+lines[1]), 0o600))

	ds := NewFileDataset(path)
	count, err := ds.Len(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, count)

	ch, errCh := ds.Samples(context.Background())
	var got []core.Sample
	for sample := range ch {
		got = append(got, sample)
	}
	for err := range errCh {
		require.NoError(t, err)
	}
	require.Len(t, got, 2)
	require.Equal(t, "x", got[0].Expected)
}
