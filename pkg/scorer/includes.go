package scorer

import (
	"context"
	"strings"

	"inspectgo/pkg/core"
)

// Includes scores responses by substring containment.
type Includes struct {
	CaseSensitive       bool
	NormalizeWhitespace bool
}

func (i Includes) Name() string {
	return "includes"
}

func (i Includes) Score(_ context.Context, sample core.Sample, response core.Response) (core.Score, error) {
	expected := normalizeText(sample.Expected, i.CaseSensitive, i.NormalizeWhitespace)
	actual := normalizeText(response.Content, i.CaseSensitive, i.NormalizeWhitespace)

	passed := strings.Contains(actual, expected)
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
