package solver

import "strings"

func applyTemplate(template string, values map[string]string) string {
	out := template
	for key, value := range values {
		out = strings.ReplaceAll(out, "{{"+key+"}}", value)
	}
	return out
}
