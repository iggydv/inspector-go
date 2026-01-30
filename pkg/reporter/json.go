package reporter

import (
	"encoding/json"
	"io"

	"inspectgo/pkg/core"
)

type JSONReporter struct {
	Writer  io.Writer
	Pretty  bool
	Compact bool
}

func (r JSONReporter) Report(report core.EvalReport) error {
	encoder := json.NewEncoder(r.Writer)
	if r.Pretty && !r.Compact {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(report)
}
