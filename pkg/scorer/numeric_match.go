package scorer

import (
	"context"
	"math"
	"regexp"
	"strconv"
	"strings"

	"inspectgo/pkg/core"
)

// NumericMatch scores responses by comparing the last number found.
type NumericMatch struct {
	Tolerance float64
}

func (n NumericMatch) Name() string {
	return "numeric"
}

func (n NumericMatch) Score(_ context.Context, sample core.Sample, response core.Response) (core.Score, error) {
	expectedNum, expectedRaw := lastNumber(sample.Expected)
	actualNum, actualRaw := lastNumber(response.Content)

	passed := false
	if expectedRaw != "" && actualRaw != "" {
		tol := n.Tolerance
		if tol <= 0 {
			tol = 1e-6
		}
		passed = math.Abs(expectedNum-actualNum) <= tol
	} else {
		expected := normalizeText(sample.Expected, false, true)
		actual := normalizeText(response.Content, false, true)
		passed = expected == actual
	}

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

var numberRegex = regexp.MustCompile(`[-+]?\d[\d,]*(?:\.\d+)?`)

func lastNumber(text string) (float64, string) {
	matches := numberRegex.FindAllString(text, -1)
	if len(matches) == 0 {
		return 0, ""
	}
	raw := matches[len(matches)-1]
	clean := strings.ReplaceAll(raw, ",", "")
	value, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0, ""
	}
	return value, raw
}
