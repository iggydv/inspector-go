package reporter

import "inspectgo/pkg/core"

// Reporter writes an evaluation report.
type Reporter interface {
	Report(report core.EvalReport) error
}

const (
	FormatJSON     = "json"
	FormatTable    = "table"
	FormatHTML     = "html"
	FormatMarkdown = "markdown"
	FormatCSV      = "csv"
)
