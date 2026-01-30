package reporter

import (
	"fmt"
	"io"

	"inspectgo/pkg/core"
)

type MarkdownReporter struct {
	Writer io.Writer
}

func (r MarkdownReporter) Report(report core.EvalReport) error {
	if _, err := fmt.Fprintf(r.Writer, "# Inspector-Go Report\n\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.Writer, "- Task: %s\n- Model: %s\n- Scorer: %s\n\n", report.TaskName, report.ModelName, report.ScorerName); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(r.Writer, "## Summary\n\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.Writer, "| Metric | Value |\n|---|---|\n"); err != nil {
		return err
	}
	lines := []struct {
		Name  string
		Value string
	}{
		{"Total samples", fmt.Sprintf("%d", report.Metrics.TotalSamples)},
		{"Success rate", fmt.Sprintf("%.2f", report.Metrics.SuccessRate)},
		{"Average score", fmt.Sprintf("%.2f", report.Metrics.AverageScore)},
		{"Median score", fmt.Sprintf("%.2f", report.Metrics.MedianScore)},
		{"P95 score", fmt.Sprintf("%.2f", report.Metrics.P95Score)},
	}
	for _, line := range lines {
		if _, err := fmt.Fprintf(r.Writer, "| %s | %s |\n", line.Name, line.Value); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(r.Writer, "\n## Samples\n\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.Writer, "| ID | Input | Expected | Output | Score | Error |\n|---|---|---|---|---|---|\n"); err != nil {
		return err
	}
	for _, result := range report.Results {
		if _, err := fmt.Fprintf(
			r.Writer,
			"| %s | %s | %s | %s | %.2f | %s |\n",
			result.Sample.ID,
			escapePipe(result.Sample.Input),
			escapePipe(result.Sample.Expected),
			escapePipe(result.Response.Content),
			result.Score.Value,
			escapePipe(result.Error),
		); err != nil {
			return err
		}
	}
	return nil
}

func escapePipe(input string) string {
	if input == "" {
		return ""
	}
	out := make([]rune, 0, len(input))
	for _, r := range input {
		if r == '|' {
			out = append(out, '\\', r)
		} else if r == '\n' || r == '\r' {
			out = append(out, ' ')
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}
