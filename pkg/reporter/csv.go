package reporter

import (
	"encoding/csv"
	"io"
	"strconv"

	"inspectgo/pkg/core"
)

type CSVReporter struct {
	Writer io.Writer
}

func (r CSVReporter) Report(report core.EvalReport) error {
	writer := csv.NewWriter(r.Writer)
	header := []string{"id", "input", "expected", "output", "score", "passed", "error", "duration_seconds"}
	if err := writer.Write(header); err != nil {
		return err
	}
	for _, result := range report.Results {
		record := []string{
			result.Sample.ID,
			result.Sample.Input,
			result.Sample.Expected,
			result.Response.Content,
			strconv.FormatFloat(result.Score.Value, 'f', 4, 64),
			strconv.FormatBool(result.Score.Passed),
			result.Error,
			strconv.FormatFloat(result.Duration.Seconds(), 'f', 6, 64),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}
