package core

// Sample is a single evaluation input and expected output.
type Sample struct {
	ID       string            `json:"id" yaml:"id"`
	Input    string            `json:"input" yaml:"input"`
	Expected string            `json:"expected" yaml:"expected"`
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}
