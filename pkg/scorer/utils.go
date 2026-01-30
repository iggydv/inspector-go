package scorer

import "strings"

func normalizeText(input string, caseSensitive bool, normalizeWhitespace bool) string {
	text := input
	if normalizeWhitespace {
		text = strings.Join(strings.Fields(text), " ")
	} else {
		text = strings.TrimSpace(text)
	}
	if !caseSensitive {
		text = strings.ToLower(text)
	}
	return text
}
