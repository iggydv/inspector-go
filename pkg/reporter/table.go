package reporter

import (
	"fmt"
	"io"

	"inspectgo/pkg/core"

	"github.com/olekukonko/tablewriter"
)

type TableReporter struct {
	Writer io.Writer
}

func (r TableReporter) Report(report core.EvalReport) error {
	table := tablewriter.NewWriter(r.Writer)
	table.Header([]string{"Metric", "Value"})
	table.Append([]string{"Total samples", fmt.Sprintf("%d", report.Metrics.TotalSamples)})
	table.Append([]string{"Success rate", fmt.Sprintf("%.2f", report.Metrics.SuccessRate)})
	table.Append([]string{"Average score", fmt.Sprintf("%.2f", report.Metrics.AverageScore)})
	table.Append([]string{"Median score", fmt.Sprintf("%.2f", report.Metrics.MedianScore)})
	table.Append([]string{"P95 score", fmt.Sprintf("%.2f", report.Metrics.P95Score)})
	table.Append([]string{"Avg latency", report.Metrics.AvgLatency.String()})
	table.Append([]string{"P95 latency", report.Metrics.P95Latency.String()})
	table.Render()
	return nil
}
