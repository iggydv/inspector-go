package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"inspectgo/pkg/core"
	"inspectgo/pkg/dataset"
	"inspectgo/pkg/inspectlog"
	"inspectgo/pkg/model"
	"inspectgo/pkg/reporter"
	"inspectgo/pkg/scorer"
	"inspectgo/pkg/solver"

	"github.com/spf13/cobra"
)

func newEvalCommand() *cobra.Command {
	var (
		datasetPath  string
		scorerName   string
		workers      int
		outputPath   string
		format       string
		modelName    string
		mockResponse string
		provider     string
		logDir       string
		logFormat    string
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
			promptTemplateResolved := ""
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

			sv := solver.BasicSolver{
				Model:          evalModel,
				Options:        core.GenerateOptions{},
				PromptTemplate: promptTemplateResolved,
			}

			eval := core.Evaluator{
				Dataset: ds,
				Solver:  sv,
				Scorer:  sc,
				Workers: workerCount,
			}

			report, err := eval.Run(context.Background())
			if err != nil {
				return err
			}
			if report.Metadata == nil {
				report.Metadata = map[string]string{}
			}
			report.Metadata["provider"] = providerResolved
			report.Metadata["solver"] = "basic"

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
	cmd.Flags().StringVar(&scorerName, "scorer", "", "scorer name (exact, includes)")
	cmd.Flags().IntVar(&workers, "workers", 0, "number of workers")
	cmd.Flags().StringVar(&outputPath, "output", "", "output file path")
	cmd.Flags().StringVar(&format, "format", "", "output format (table, json, html, markdown, csv)")
	cmd.Flags().StringVar(&modelName, "model", "", "model name (mock)")
	cmd.Flags().StringVar(&mockResponse, "mock-response", "", "fixed mock response")
	cmd.Flags().StringVar(&provider, "provider", "", "model provider (mock, openai)")
	cmd.Flags().StringVar(&logDir, "log-dir", "", "directory for Inspect-compatible logs")
	cmd.Flags().StringVar(&logFormat, "log-format", "", "log format (inspect-eval, inspect-json, none)")

	return cmd
}

func buildScorer(name string) (core.Scorer, error) {
	switch name {
	case "exact":
		return scorer.ExactMatch{CaseSensitive: false, NormalizeWhitespace: true}, nil
	case "includes":
		return scorer.Includes{CaseSensitive: false, NormalizeWhitespace: true}, nil
	default:
		return nil, fmt.Errorf("unknown scorer: %s", name)
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
