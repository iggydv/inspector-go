 # InspectGo
 
 InspectGo is a Go-based evaluation framework for LLMs.
 
 ## Local MVP
 
 This repository contains a minimal local MVP that runs evaluation with:
 - JSON/JSONL datasets
 - Exact match and includes scorers
 - A mock model for offline testing
 - A CLI with `eval` and `list` commands
 
 ### Quick start
 
 ```sh
 go run ./cmd/inspectgo list
 go run ./cmd/inspectgo eval --dataset ./examples/math/dataset.jsonl --scorer exact
 ```

### Reporting formats

```sh
go run ./cmd/inspectgo eval --dataset ./examples/math/dataset.jsonl --format json
go run ./cmd/inspectgo eval --dataset ./examples/math/dataset.jsonl --format table
go run ./cmd/inspectgo eval --dataset ./examples/math/dataset.jsonl --format html --output ./report.html
go run ./cmd/inspectgo eval --dataset ./examples/math/dataset.jsonl --format markdown --output ./report.md
go run ./cmd/inspectgo eval --dataset ./examples/math/dataset.jsonl --format csv --output ./report.csv
```

### OpenAI provider

```sh
export OPENAI_API_KEY="your-key"
go run ./cmd/inspectgo eval --provider openai --model gpt-4o-mini --dataset ./examples/math/dataset.jsonl
```

### Anthropic provider

```sh
export ANTHROPIC_API_KEY="your-key"
go run ./cmd/inspectgo eval --provider anthropic --model claude-3-5-haiku-latest --dataset ./examples/math/dataset.jsonl
```

### Inspect-compatible logs

The CLI can emit Inspect-style JSON logs for use with Inspect tools:

```sh
go run ./cmd/inspectgo eval --dataset ./examples/math/dataset.jsonl --log-format inspect-eval --log-dir ./logs
```
 
