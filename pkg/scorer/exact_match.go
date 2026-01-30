package scorer

import (
	"context"

	"inspectgo/pkg/core"
)

// ExactMatch scores responses by exact string match.
type ExactMatch struct {
	CaseSensitive       bool
	NormalizeWhitespace bool
}

func (e ExactMatch) Name() string {
	return "exact"
}

func (e ExactMatch) Score(_ context.Context, sample core.Sample, response core.Response) (core.Score, error) {
	expected := normalizeText(sample.Expected, e.CaseSensitive, e.NormalizeWhitespace)
	actual := normalizeText(response.Content, e.CaseSensitive, e.NormalizeWhitespace)

	passed := expected == actual
	value := 0.0
	if passed {
		value = 1
	}
	return core.Score{
		Value:  value,
		Max:    1,
		Passed: passed,
	}, nil
}
