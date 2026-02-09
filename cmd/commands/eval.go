package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"inspectgo/pkg/core"
	"inspectgo/pkg/dataset"
	"inspectgo/pkg/inspectlog"
	"inspectgo/pkg/model"
	"inspectgo/pkg/reporter"
	"inspectgo/pkg/scorer"
	"inspectgo/pkg/solver"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

func newEvalCommand() *cobra.Command {
	var (
		datasetPath    string
		scorerName     string
		workers        int
		outputPath     string
		format         string
		modelName      string
		mockResponse   string
		provider       string
		fewshotCount   int
		rateLimitRPS   float64
		rateLimitBurst int
		promptTemplate string
		logDir         string
		logFormat      string
		solverName     string
		temperature    float64
		maxTokens      int
		topP           float64
		sampleTimeout  time.Duration
		maxTotalTokens int
	)

	cmd := &cobra.Command{
		Use:   "eval",
		Short: "Run an evaluation",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := resolveString(datasetPath, appConfig.Dataset)
			if path == "" {
				return errors.New("dataset path is required")
			}
			scorerNameResolved := resolveString(scorerName, appConfig.Scorer)
			if scorerNameResolved == "" {
				scorerNameResolved = "exact"
			}
			formatResolved := resolveString(format, appConfig.Format)
			if formatResolved == "" {
				formatResolved = "table"
			}
			outputResolved := resolveString(outputPath, appConfig.Output)
			modelResolved := resolveString(modelName, appConfig.Model.Name)
			mockResolved := resolveString(mockResponse, appConfig.Model.MockResponse)
			providerResolved := resolveString(provider, appConfig.Provider)
			if providerResolved == "" {
				providerResolved = "mock"
			}
			promptTemplateResolved := promptTemplate
			if modelResolved == "" {
				switch providerResolved {
				case "openai":
					modelResolved = "gpt-4o-mini"
				case "anthropic":
					modelResolved = "claude-3-5-haiku-latest"
				default:
					modelResolved = "mock"
				}
			}
			logDirResolved := resolveString(logDir, appConfig.LogDir)
			logFormatResolved := resolveString(logFormat, appConfig.LogFormat)
			if logFormatResolved == "" {
				logFormatResolved = "inspect-eval"
			}
			workerCount := resolveInt(workers, appConfig.Workers, 1)

			ds := dataset.NewFileDataset(path)
			sc, err := buildScorer(scorerNameResolved)
			if err != nil {
				return err
			}

			totalSamples := 0
			if count, err := ds.Len(context.Background()); err == nil {
				totalSamples = count
			}
			progress := newProgressBar(progressWriter(cmd), totalSamples)
			progress.Update(0, 0)

			var rateLimiter core.RateLimiter
			stopLimiter := func() {}
			if rateLimitRPS > 0 {
				limiter, stop, err := core.NewRateLimiter(rateLimitRPS, rateLimitBurst)
				if err != nil {
					return err
				}
				rateLimiter = limiter
				stopLimiter = stop
				defer stopLimiter()
			}

			var evalModel core.Model
			switch providerResolved {
			case "mock":
				evalModel = model.MockModel{
					NameValue:    modelResolved,
					ResponseText: mockResolved,
				}
			case "openai":
				openaiModel, err := model.NewOpenAIModelFromEnv(modelResolved)
				if err != nil {
					return err
				}
				openaiCfg := appConfig.OpenAI
				if openaiCfg.Model != "" && modelResolved == "gpt-4o-mini" {
					openaiModel.Model = openaiCfg.Model
				}
				if openaiCfg.TimeoutSeconds > 0 {
					openaiModel.Timeout = time.Duration(openaiCfg.TimeoutSeconds) * time.Second
				}
				if openaiCfg.MaxRetries > 0 {
					openaiModel.MaxRetries = openaiCfg.MaxRetries
				}
				if openaiCfg.BackoffMillis > 0 {
					openaiModel.Backoff = time.Duration(openaiCfg.BackoffMillis) * time.Millisecond
				}
				evalModel = openaiModel
			case "anthropic":
				anthropicModel, err := model.NewAnthropicModelFromEnv(modelResolved)
				if err != nil {
					return err
				}
				anthropicCfg := appConfig.Anthropic
				if anthropicCfg.Model != "" && modelResolved == "claude-3-5-haiku-latest" {
					anthropicModel.Model = anthropicCfg.Model
				}
				if anthropicCfg.TimeoutSeconds > 0 {
					anthropicModel.Timeout = time.Duration(anthropicCfg.TimeoutSeconds) * time.Second
				}
				if anthropicCfg.MaxRetries > 0 {
					anthropicModel.MaxRetries = anthropicCfg.MaxRetries
				}
				if anthropicCfg.BackoffMillis > 0 {
					anthropicModel.Backoff = time.Duration(anthropicCfg.BackoffMillis) * time.Millisecond
				}
				if anthropicCfg.MaxTokens > 0 {
					anthropicModel.MaxTokens = anthropicCfg.MaxTokens
				}
				evalModel = anthropicModel
			default:
				return fmt.Errorf("unknown provider: %s", providerResolved)
			}

			opts := core.GenerateOptions{
				Temperature: float32(temperature),
				MaxTokens:   maxTokens,
				TopP:        float32(topP),
			}

			sv, err := buildSolver(solverName, evalModel, opts, promptTemplateResolved, fewshotCount, ds)
			if err != nil {
				return err
			}

			eval := core.Evaluator{
				Dataset:        ds,
				Solver:         sv,
				Scorer:         sc,
				Workers:        workerCount,
				TotalSamples:   totalSamples,
				SampleTimeout:  sampleTimeout,
				MaxTotalTokens: maxTotalTokens,
				Progress: func(completed, total, inflight int) {
					progress.Update(completed, inflight)
				},
				RateLimiter: rateLimiter,
			}

			report, err := eval.Run(context.Background())
			if err != nil {
				return err
			}
			if report.Metadata == nil {
				report.Metadata = map[string]string{}
			}
			report.Metadata["provider"] = providerResolved
			report.Metadata["solver"] = sv.Name()

			writer := os.Stdout
			if outputResolved != "" {
				file, err := os.Create(outputResolved)
				if err != nil {
					return err
				}
				defer file.Close()
				writer = file
			}

			rep, err := buildReporter(formatResolved, writer)
			if err != nil {
				return err
			}

			if err := rep.Report(report); err != nil {
				return err
			}

			if logFormatResolved != "none" {
				if logDirResolved == "" {
					logDirResolved = "./logs"
				}
				if err := writeInspectLog(logFormatResolved, logDirResolved, report); err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&datasetPath, "dataset", "", "path to dataset file")
	cmd.Flags().StringVar(&scorerName, "scorer", "", "scorer name (exact, includes, numeric)")
	cmd.Flags().IntVar(&workers, "workers", 0, "number of workers")
	cmd.Flags().StringVar(&outputPath, "output", "", "output file path")
	cmd.Flags().StringVar(&format, "format", "", "output format (table, json, html, markdown, csv)")
	cmd.Flags().StringVar(&modelName, "model", "", "model name (mock)")
	cmd.Flags().StringVar(&mockResponse, "mock-response", "", "fixed mock response")
	cmd.Flags().StringVar(&provider, "provider", "", "model provider (mock, openai)")
	cmd.Flags().IntVar(&fewshotCount, "fewshot", 0, "number of few-shot examples")
	cmd.Flags().Float64Var(&rateLimitRPS, "rate-limit-rps", 0, "max requests per second (0 = unlimited)")
	cmd.Flags().IntVar(&rateLimitBurst, "rate-limit-burst", 1, "rate limit burst size")
	cmd.Flags().StringVar(&promptTemplate, "prompt-template", "", "prompt template with {{input}} placeholder")
	cmd.Flags().StringVar(&logDir, "log-dir", "", "directory for Inspect-compatible logs")
	cmd.Flags().StringVar(&logFormat, "log-format", "", "log format (inspect-eval, inspect-json, none)")
	cmd.Flags().StringVar(&solverName, "solver", "", "solver name (basic, chain-of-thought, cot, few-shot, multi-step, self-consistency, self-critique); comma-separated for chaining")
	cmd.Flags().Float64Var(&temperature, "temperature", 0, "model temperature (0 = default)")
	cmd.Flags().IntVar(&maxTokens, "max-tokens", 0, "max completion tokens (0 = solver default)")
	cmd.Flags().Float64Var(&topP, "top-p", 0, "nucleus sampling top-p (0 = default)")
	cmd.Flags().DurationVar(&sampleTimeout, "sample-timeout", 60*time.Second, "per-sample timeout")
	cmd.Flags().IntVar(&maxTotalTokens, "max-total-tokens", 0, "max total tokens budget (0 = unlimited)")

	return cmd
}

func buildScorer(name string) (core.Scorer, error) {
	switch name {
	case "exact":
		return scorer.ExactMatch{CaseSensitive: false, NormalizeWhitespace: true}, nil
	case "includes":
		return scorer.Includes{CaseSensitive: false, NormalizeWhitespace: true}, nil
	case "numeric":
		return scorer.NumericMatch{}, nil
	default:
		return nil, fmt.Errorf("unknown scorer: %s", name)
	}
}

func buildSolver(name string, m core.Model, opts core.GenerateOptions, promptTemplate string, fewshotCount int, ds core.Dataset) (core.Solver, error) {
	// When --solver is empty, fall back to existing behavior
	if name == "" {
		if fewshotCount > 0 {
			examples, err := loadFewShotExamples(context.Background(), ds, fewshotCount)
			if err != nil {
				return nil, err
			}
			return solver.FewShotSolver{
				Model:          m,
				Options:        opts,
				Examples:       examples,
				PromptTemplate: promptTemplate,
			}, nil
		}
		return solver.BasicSolver{
			Model:          m,
			Options:        opts,
			PromptTemplate: promptTemplate,
		}, nil
	}

	// Handle comma-separated solver chaining
	parts := strings.Split(name, ",")
	if len(parts) > 1 {
		solvers := make([]core.Solver, 0, len(parts))
		for i, part := range parts {
			part = strings.TrimSpace(part)
			s, err := buildSingleSolver(part, m, opts, promptTemplate, fewshotCount, ds, i > 0)
			if err != nil {
				return nil, err
			}
			solvers = append(solvers, s)
		}
		return solver.PipelineSolver{Solvers: solvers}, nil
	}

	return buildSingleSolver(name, m, opts, promptTemplate, fewshotCount, ds, false)
}

func buildSingleSolver(name string, m core.Model, opts core.GenerateOptions, promptTemplate string, fewshotCount int, ds core.Dataset, chained bool) (core.Solver, error) {
	switch name {
	case "basic":
		return solver.BasicSolver{
			Model:          m,
			Options:        opts,
			PromptTemplate: promptTemplate,
		}, nil
	case "chain-of-thought", "cot":
		return solver.ChainOfThoughtSolver{
			Model:          m,
			Options:        opts,
			PromptTemplate: promptTemplate,
			ExtractAnswer:  true,
		}, nil
	case "few-shot":
		examples, err := loadFewShotExamples(context.Background(), ds, fewshotCount)
		if err != nil {
			return nil, err
		}
		return solver.FewShotSolver{
			Model:          m,
			Options:        opts,
			Examples:       examples,
			PromptTemplate: promptTemplate,
		}, nil
	case "multi-step":
		return solver.MultiStepSolver{
			Model:   m,
			Options: opts,
		}, nil
	case "self-consistency":
		return solver.SelfConsistencySolver{
			Model:          m,
			Options:        opts,
			PromptTemplate: promptTemplate,
		}, nil
	case "self-critique":
		return solver.SelfCritiqueSolver{
			Model:       m,
			Options:     opts,
			SkipInitial: chained,
		}, nil
	default:
		return nil, fmt.Errorf("unknown solver: %s", name)
	}
}

func buildReporter(format string, writer io.Writer) (reporter.Reporter, error) {
	switch format {
	case reporter.FormatJSON:
		return reporter.JSONReporter{Writer: writer, Pretty: true}, nil
	case reporter.FormatTable:
		return reporter.TableReporter{Writer: writer}, nil
	case reporter.FormatHTML:
		return reporter.HTMLReporter{Writer: writer}, nil
	case reporter.FormatMarkdown:
		return reporter.MarkdownReporter{Writer: writer}, nil
	case reporter.FormatCSV:
		return reporter.CSVReporter{Writer: writer}, nil
	default:
		return nil, fmt.Errorf("unknown format: %s", format)
	}
}

func writeInspectLog(format string, logDir string, report core.EvalReport) error {
	switch format {
	case "inspect", "inspect-eval", "eval":
		log := inspectlog.FromReport(report)
		_, err := inspectlog.WriteEval(logDir, log)
		return err
	case "inspect-json":
		log := inspectlog.FromReport(report)
		_, err := inspectlog.WriteJSON(logDir, log)
		return err
	case "none":
		return nil
	default:
		return fmt.Errorf("unknown log format: %s", format)
	}
}

func loadFewShotExamples(ctx context.Context, ds core.Dataset, count int) ([]solver.FewShotExample, error) {
	if count <= 0 {
		return nil, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sampleCh, errCh := ds.Samples(ctx)
	examples := make([]solver.FewShotExample, 0, count)
	reachedCount := false

	for {
		if !reachedCount && sampleCh == nil {
			return nil, errors.New("few-shot: dataset returned no samples")
		}
		if sampleCh == nil && errCh == nil {
			break
		}

		select {
		case sample, ok := <-sampleCh:
			if !ok {
				sampleCh = nil
				continue
			}
			examples = append(examples, solver.FewShotExample{
				Input:  sample.Input,
				Output: sample.Expected,
			})
			if len(examples) >= count {
				reachedCount = true
				cancel()
			}
		case err, ok := <-errCh:
			if ok && err != nil {
				if errors.Is(err, context.Canceled) && reachedCount {
					errCh = nil
					continue
				}
				return nil, err
			}
			if !ok {
				errCh = nil
			}
		}
	}

	return examples, nil
}

type progressBar struct {
	writer io.Writer
	total  int
	start  time.Time
	isTTY  bool
}

func newProgressBar(writer io.Writer, total int) *progressBar {
	return &progressBar{
		writer: writer,
		total:  total,
		start:  time.Now(),
		isTTY:  isTerminal(writer),
	}
}

func (p *progressBar) Update(completed int, inflight int) {
	width := 30
	if p.total <= 0 {
		elapsed := time.Since(p.start).Truncate(time.Second)
		if p.isTTY {
			fmt.Fprintf(p.writer, "\rProcessed %d samples (inflight %d) (%s)", completed, inflight, elapsed)
		} else {
			fmt.Fprintf(p.writer, "Processed %d samples (inflight %d) (%s)\n", completed, inflight, elapsed)
		}
		return
	}

	ratio := float64(completed) / float64(p.total)
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(width))

	bar := strings.Repeat("=", filled) + strings.Repeat(".", width-filled)
	percent := int(ratio * 100)
	elapsed := time.Since(p.start).Truncate(time.Second)

	barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
	line := fmt.Sprintf("[%s] %3d%% (%d/%d) inflight %d %s", barStyle.Render(bar), percent, completed, p.total, inflight, elapsed)
	if p.isTTY {
		fmt.Fprintf(p.writer, "\r%s", line)
	} else {
		fmt.Fprintf(p.writer, "%s\n", line)
	}

	if completed >= p.total {
		fmt.Fprintln(p.writer)
	}
}

func isTerminal(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	fd := file.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

func progressWriter(cmd *cobra.Command) io.Writer {
	stderr := cmd.ErrOrStderr()
	stdout := cmd.OutOrStdout()

	if isTerminal(stderr) {
		return stderr
	}
	if isTerminal(stdout) {
		return stdout
	}
	return stderr
}

func resolveString(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func resolveInt(value int, fallback int, defaultValue int) int {
	if value > 0 {
		return value
	}
	if fallback > 0 {
		return fallback
	}
	return defaultValue
}
