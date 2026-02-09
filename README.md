# Inspector-Go

Inspector-Go is a Go-based evaluation framework for LLMs.

## Quick start

```sh
go run ./cmd/inspectgo list
go run ./cmd/inspectgo eval --dataset ./examples/math/dataset.jsonl --scorer exact
```

## CLI Reference

### `eval` Command

```sh
go run ./cmd/inspectgo eval [flags]
```

#### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dataset` | `string` | | Path to dataset file (JSONL/JSON) |
| `--scorer` | `string` | `exact` | Scorer name (`exact`, `includes`, `numeric`) |
| `--solver` | `string` | | Solver name; comma-separated for chaining (see [Solvers](#solvers)) |
| `--provider` | `string` | `mock` | Model provider (`mock`, `openai`, `anthropic`) |
| `--model` | `string` | auto | Model name (defaults per provider) |
| `--temperature` | `float` | `0` | Model sampling temperature (`0` = provider default) |
| `--max-tokens` | `int` | `0` | Max completion tokens (`0` = solver default) |
| `--top-p` | `float` | `0` | Nucleus sampling top-p (`0` = provider default) |
| `--workers` | `int` | `1` | Number of concurrent workers |
| `--sample-timeout` | `duration` | `60s` | Per-sample timeout |
| `--max-total-tokens` | `int` | `0` | Total token budget; eval stops when exceeded (`0` = unlimited) |
| `--fewshot` | `int` | `0` | Number of few-shot examples to prepend |
| `--prompt-template` | `string` | | Custom prompt template with `{{input}}` placeholder |
| `--mock-response` | `string` | | Fixed response for mock provider |
| `--rate-limit-rps` | `float` | `0` | Max requests per second (`0` = unlimited) |
| `--rate-limit-burst` | `int` | `1` | Rate limit burst size |
| `--format` | `string` | `table` | Output format (`table`, `json`, `html`, `markdown`, `csv`) |
| `--output` | `string` | stdout | Output file path |
| `--log-dir` | `string` | `./logs` | Directory for Inspect-compatible logs |
| `--log-format` | `string` | `inspect-eval` | Log format (`inspect-eval`, `inspect-json`, `none`) |

### Solvers

| Name | Aliases | Description | Default MaxTokens |
|------|---------|-------------|-------------------|
| `basic` | | Single prompt, returns final answer | 256 |
| `chain-of-thought` | `cot` | Step-by-step reasoning with answer extraction | 1024 |
| `few-shot` | | Prepends example Q&A pairs from the dataset | 256 |
| `multi-step` | | Multiple sequential generation steps | |
| `self-consistency` | | Samples N responses in parallel, picks majority answer | |
| `self-critique` | | 3-step: initial answer, critique, revision | 256 (revision) |

Solvers can be chained with commas. When `self-critique` appears after another solver in a chain, it automatically skips the initial answer step and critiques the previous solver's output.

```sh
# Single solver
--solver chain-of-thought

# Chained: chain-of-thought -> self-critique
--solver chain-of-thought,self-critique
```

### Scorers

| Name | Description |
|------|-------------|
| `exact` | Case-insensitive exact match with whitespace normalization |
| `includes` | Case-insensitive substring match with whitespace normalization |
| `numeric` | Numeric equality comparison |

## Providers

### Mock

```sh
go run ./cmd/inspectgo eval --dataset ./examples/math/dataset.jsonl --scorer exact
go run ./cmd/inspectgo eval --dataset ./examples/math/dataset.jsonl --mock-response "42"
```

### OpenAI

```sh
export OPENAI_API_KEY="your-key"
go run ./cmd/inspectgo eval --provider openai --model gpt-4o-mini --dataset ./examples/math/dataset.jsonl
```

### Anthropic

```sh
export ANTHROPIC_API_KEY="your-key"
go run ./cmd/inspectgo eval --provider anthropic --model claude-3-5-haiku-latest --dataset ./examples/math/dataset.jsonl
```

## Examples

```sh
# Basic eval with exact scorer
go run ./cmd/inspectgo eval --dataset ./examples/math/dataset.jsonl --scorer exact

# Chain-of-thought with answer extraction
go run ./cmd/inspectgo eval --dataset ./examples/math/dataset.jsonl --solver cot --provider openai --scorer numeric

# Chain-of-thought piped into self-critique
go run ./cmd/inspectgo eval --dataset ./examples/math/dataset.jsonl --solver cot,self-critique --provider openai --scorer numeric

# Parallel workers with rate limiting and temperature control
go run ./cmd/inspectgo eval --dataset ./examples/math/dataset.jsonl --solver cot --provider openai --workers 5 --rate-limit-rps 10 --temperature 0.7

# Token budget guard (stop after 100k tokens)
go run ./cmd/inspectgo eval --dataset ./examples/math/dataset.jsonl --provider openai --max-total-tokens 100000

# Output as JSON with Inspect-compatible logs
go run ./cmd/inspectgo eval --dataset ./examples/math/dataset.jsonl --format json --output report.json --log-dir ./logs
```

## Reporting Formats

```sh
go run ./cmd/inspectgo eval --dataset ./examples/math/dataset.jsonl --format table
go run ./cmd/inspectgo eval --dataset ./examples/math/dataset.jsonl --format json
go run ./cmd/inspectgo eval --dataset ./examples/math/dataset.jsonl --format html --output report.html
go run ./cmd/inspectgo eval --dataset ./examples/math/dataset.jsonl --format markdown --output report.md
go run ./cmd/inspectgo eval --dataset ./examples/math/dataset.jsonl --format csv --output report.csv
```

## Inspect-Compatible Logs

The CLI emits Inspect-style JSON logs for use with Inspect tools:

```sh
go run ./cmd/inspectgo eval --dataset ./examples/math/dataset.jsonl --log-format inspect-eval --log-dir ./logs
```

These `.eval` logs are compatible with the Inspect-AI VS Code extension and the
Inspect-AI CLI log tools (`inspect view`, `inspect log list`, `inspect log dump`).

## Package Reference

### `core`

| Type | Description |
|------|-------------|
| `Model` | Interface: `Name() string`, `Generate(ctx, prompt, GenerateOptions) (Response, error)` |
| `Solver` | Interface: `Name() string`, `Solve(ctx, Sample) (Response, error)` |
| `Scorer` | Interface: `Name() string`, `Score(ctx, Sample, Response) (Score, error)` |
| `Dataset` | Interface: streams `Sample` values for evaluation |
| `Evaluator` | Runs a dataset through a solver and scorer with concurrent workers |
| `GenerateOptions` | Controls model generation: `Temperature`, `MaxTokens`, `TopP`, `Stop`, `SystemPrompt` |
| `Response` | Model response with `Content`, `TokenUsage`, `Latency` |
| `Sample` | Evaluation input: `ID`, `Input`, `Expected`, `Metadata` |

### `solver`

| Type | Description |
|------|-------------|
| `BasicSolver` | Single prompt with system prompt for concise answers |
| `ChainOfThoughtSolver` | Step-by-step reasoning with optional answer extraction via `ExtractAnswer` |
| `FewShotSolver` | Prepends example Q&A pairs before the prompt |
| `MultiStepSolver` | Sequential multi-step generation with token aggregation |
| `SelfConsistencySolver` | Parallel N-sample majority vote |
| `SelfCritiqueSolver` | 3-step: initial answer, critique, revision; `SkipInitial` for chaining |
| `PipelineSolver` | Composes solvers sequentially; output of each becomes input to next |
| `ToolUseSolver` | Runs a tool call before prompting the model |
| `ExtractFinalAnswer(string) string` | Extracts clean answer from reasoning text using pattern matching |
