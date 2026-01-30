package reporter

import (
	"html/template"
	"io"

	"inspectgo/pkg/core"
)

type HTMLReporter struct {
	Writer io.Writer
	Title  string
}

func (r HTMLReporter) Report(report core.EvalReport) error {
	title := r.Title
	if title == "" {
		title = "InspectGo Report"
	}

	data := struct {
		Title  string
		Report core.EvalReport
	}{
		Title:  title,
		Report: report,
	}

	tpl := template.Must(template.New("report").Parse(htmlTemplate))
	return tpl.Execute(r.Writer, data)
}

const htmlTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{ .Title }}</title>
  <style>
    body { font-family: Arial, sans-serif; margin: 24px; }
    table { border-collapse: collapse; width: 100%; margin-top: 16px; }
    th, td { border: 1px solid #ddd; padding: 8px; }
    th { background: #f5f5f5; text-align: left; }
    .meta { margin-bottom: 12px; }
  </style>
</head>
<body>
  <h1>{{ .Title }}</h1>
  <div class="meta">
    <div><strong>Task:</strong> {{ .Report.TaskName }}</div>
    <div><strong>Model:</strong> {{ .Report.ModelName }}</div>
    <div><strong>Scorer:</strong> {{ .Report.ScorerName }}</div>
  </div>
  <h2>Summary</h2>
  <table>
    <tr><th>Metric</th><th>Value</th></tr>
    <tr><td>Total samples</td><td>{{ .Report.Metrics.TotalSamples }}</td></tr>
    <tr><td>Success rate</td><td>{{ printf "%.2f" .Report.Metrics.SuccessRate }}</td></tr>
    <tr><td>Average score</td><td>{{ printf "%.2f" .Report.Metrics.AverageScore }}</td></tr>
    <tr><td>Median score</td><td>{{ printf "%.2f" .Report.Metrics.MedianScore }}</td></tr>
    <tr><td>P95 score</td><td>{{ printf "%.2f" .Report.Metrics.P95Score }}</td></tr>
  </table>
  <h2>Samples</h2>
  <table>
    <tr><th>ID</th><th>Input</th><th>Expected</th><th>Output</th><th>Score</th><th>Error</th></tr>
    {{ range .Report.Results }}
    <tr>
      <td>{{ .Sample.ID }}</td>
      <td>{{ .Sample.Input }}</td>
      <td>{{ .Sample.Expected }}</td>
      <td>{{ .Response.Content }}</td>
      <td>{{ printf "%.2f" .Score.Value }}</td>
      <td>{{ .Error }}</td>
    </tr>
    {{ end }}
  </table>
</body>
</html>
`
