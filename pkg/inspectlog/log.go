package inspectlog

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"inspectgo/pkg/core"
)

type EvalLog struct {
	Version     int                   `json:"version"`
	Status      string                `json:"status"`
	Eval        EvalSpec              `json:"eval"`
	Plan        EvalPlan              `json:"plan"`
	Results     *EvalResults          `json:"results,omitempty"`
	Stats       EvalStats             `json:"stats"`
	Error       *EvalError            `json:"error,omitempty"`
	Invalidated bool                  `json:"invalidated,omitempty"`
	Samples     []EvalSample          `json:"samples,omitempty"`
	Reductions  []EvalSampleReduction `json:"reductions,omitempty"`
}

type EvalError struct {
	Message       string `json:"message"`
	Traceback     string `json:"traceback"`
	TracebackANSI string `json:"traceback_ansi"`
}

type EvalSpec struct {
	Created  string         `json:"created"`
	Task     string         `json:"task"`
	Dataset  EvalDataset    `json:"dataset"`
	Model    string         `json:"model"`
	Config   EvalConfig     `json:"config"`
	TaskArgs map[string]any `json:"task_args,omitempty"`

	TaskID      string `json:"task_id,omitempty"`
	TaskVersion int    `json:"task_version,omitempty"`
	RunID       string `json:"run_id,omitempty"`
	EvalID      string `json:"eval_id,omitempty"`
}

type EvalDataset struct {
	Name      string   `json:"name,omitempty"`
	Location  string   `json:"location,omitempty"`
	Samples   *int     `json:"samples,omitempty"`
	SampleIDs []string `json:"sample_ids,omitempty"`
	Shuffled  *bool    `json:"shuffled,omitempty"`
}

type EvalConfig struct {
	Epochs      *int  `json:"epochs,omitempty"`
	LogSamples  *bool `json:"log_samples,omitempty"`
	LogRealtime *bool `json:"log_realtime,omitempty"`
	LogImages   *bool `json:"log_images,omitempty"`
}

type EvalPlan struct {
	Name   string         `json:"name,omitempty"`
	Steps  []EvalPlanStep `json:"steps,omitempty"`
	Finish *EvalPlanStep  `json:"finish,omitempty"`
	Config map[string]any `json:"config,omitempty"`
}

type EvalPlanStep struct {
	Solver       string         `json:"solver"`
	Params       map[string]any `json:"params,omitempty"`
	ParamsPassed map[string]any `json:"params_passed,omitempty"`
}

type EvalResults struct {
	TotalSamples     int         `json:"total_samples"`
	CompletedSamples int         `json:"completed_samples"`
	Scores           []EvalScore `json:"scores,omitempty"`
}

type EvalScore struct {
	Name            string                `json:"name"`
	Scorer          string                `json:"scorer"`
	Reducer         string                `json:"reducer,omitempty"`
	ScoredSamples   *int                  `json:"scored_samples,omitempty"`
	UnscoredSamples *int                  `json:"unscored_samples,omitempty"`
	Params          map[string]any        `json:"params,omitempty"`
	Metrics         map[string]EvalMetric `json:"metrics,omitempty"`
	Metadata        map[string]any        `json:"metadata,omitempty"`
}

type EvalMetric struct {
	Name     string         `json:"name"`
	Value    float64        `json:"value"`
	Params   map[string]any `json:"params,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type EvalStats struct {
	StartedAt   string                `json:"started_at"`
	CompletedAt string                `json:"completed_at"`
	ModelUsage  map[string]ModelUsage `json:"model_usage,omitempty"`
}

type ModelUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type EvalSample struct {
	ID          string                `json:"id"`
	Epoch       int                   `json:"epoch"`
	Input       string                `json:"input"`
	Choices     []string              `json:"choices,omitempty"`
	Target      string                `json:"target"`
	Output      ModelOutput           `json:"output"`
	Scores      map[string]Score      `json:"scores,omitempty"`
	Metadata    map[string]any        `json:"metadata,omitempty"`
	ModelUsage  map[string]ModelUsage `json:"model_usage,omitempty"`
	StartedAt   string                `json:"started_at,omitempty"`
	CompletedAt string                `json:"completed_at,omitempty"`
	TotalTime   float64               `json:"total_time,omitempty"`
	WorkingTime float64               `json:"working_time,omitempty"`
	UUID        string                `json:"uuid,omitempty"`
	Error       *EvalError            `json:"error,omitempty"`
}

type ModelOutput struct {
	Model      string      `json:"model,omitempty"`
	Completion string      `json:"completion,omitempty"`
	Usage      *ModelUsage `json:"usage,omitempty"`
	Time       *float64    `json:"time,omitempty"`
}

type Score struct {
	Value any `json:"value"`
}

type EvalSampleSummary struct {
	ID          string                `json:"id"`
	Epoch       int                   `json:"epoch"`
	Input       string                `json:"input"`
	Target      string                `json:"target"`
	Metadata    map[string]any        `json:"metadata,omitempty"`
	Scores      map[string]Score      `json:"scores,omitempty"`
	ModelUsage  map[string]ModelUsage `json:"model_usage,omitempty"`
	TotalTime   float64               `json:"total_time,omitempty"`
	WorkingTime float64               `json:"working_time,omitempty"`
	Error       string                `json:"error,omitempty"`
	Completed   bool                  `json:"completed,omitempty"`
}

type EvalSampleReduction struct {
	Scorer  string        `json:"scorer"`
	Reducer string        `json:"reducer,omitempty"`
	Samples []SampleScore `json:"samples"`
}

type SampleScore struct {
	SampleID string `json:"sample_id,omitempty"`
	Score    Score  `json:"score"`
}

type LogStart struct {
	Version int      `json:"version"`
	Eval    EvalSpec `json:"eval"`
	Plan    EvalPlan `json:"plan"`
}

func FromReport(report core.EvalReport) EvalLog {
	scoreName := report.ScorerName
	if scoreName == "" {
		scoreName = "score"
	}

	startedAt := report.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	completedAt := report.FinishedAt
	if completedAt.IsZero() {
		completedAt = time.Now().UTC()
	}

	sampleIDs := make([]string, 0, len(report.Results))
	for idx, result := range report.Results {
		id := result.Sample.ID
		if id == "" {
			id = fmt.Sprintf("%d", idx+1)
		}
		sampleIDs = append(sampleIDs, id)
	}

	totalSamples := len(report.Results)
	completedSamples := 0
	for _, result := range report.Results {
		if result.Error == "" {
			completedSamples++
		}
	}

	usage := ModelUsage{
		InputTokens:  report.Metrics.TokenUsage.PromptTokens,
		OutputTokens: report.Metrics.TokenUsage.CompletionTokens,
		TotalTokens:  report.Metrics.TokenUsage.TotalTokens,
	}

	epochs := 1
	logSamples := true
	logRealtime := false
	logImages := false

	metrics := map[string]EvalMetric{
		"success_rate":  {Name: "success_rate", Value: report.Metrics.SuccessRate},
		"average_score": {Name: "average_score", Value: report.Metrics.AverageScore},
		"median_score":  {Name: "median_score", Value: report.Metrics.MedianScore},
		"p95_score":     {Name: "p95_score", Value: report.Metrics.P95Score},
		"p99_score":     {Name: "p99_score", Value: report.Metrics.P99Score},
	}

	score := EvalScore{
		Name:            scoreName,
		Scorer:          scoreName,
		ScoredSamples:   &totalSamples,
		UnscoredSamples: intPointer(totalSamples - completedSamples),
		Metrics:         metrics,
	}

	results := &EvalResults{
		TotalSamples:     totalSamples,
		CompletedSamples: completedSamples,
		Scores:           []EvalScore{score},
	}

	samples := make([]EvalSample, 0, len(report.Results))
	summaries := make([]EvalSampleSummary, 0, len(report.Results))

	for idx, result := range report.Results {
		id := result.Sample.ID
		if id == "" {
			id = fmt.Sprintf("%d", idx+1)
		}

		modelUsage := map[string]ModelUsage{}
		if report.ModelName != "" {
			modelUsage[report.ModelName] = usage
		}

		outputUsage := usage
		outputTime := result.Response.Latency.Seconds()
		sampleScore := map[string]Score{
			scoreName: {Value: result.Score.Value},
		}

		var sampleError *EvalError
		if result.Error != "" {
			sampleError = &EvalError{
				Message:       result.Error,
				Traceback:     "",
				TracebackANSI: "",
			}
		}

		sample := EvalSample{
			ID:          id,
			Epoch:       1,
			Input:       result.Sample.Input,
			Target:      result.Sample.Expected,
			Output:      ModelOutput{Model: report.ModelName, Completion: result.Response.Content, Usage: &outputUsage, Time: &outputTime},
			Scores:      sampleScore,
			Metadata:    mapStringStringToAny(result.Sample.Metadata),
			ModelUsage:  modelUsage,
			TotalTime:   result.Duration.Seconds(),
			WorkingTime: result.Duration.Seconds(),
			Error:       sampleError,
		}
		samples = append(samples, sample)

		summary := EvalSampleSummary{
			ID:          id,
			Epoch:       1,
			Input:       result.Sample.Input,
			Target:      result.Sample.Expected,
			Metadata:    mapStringStringToAny(result.Sample.Metadata),
			Scores:      sampleScore,
			ModelUsage:  modelUsage,
			TotalTime:   result.Duration.Seconds(),
			WorkingTime: result.Duration.Seconds(),
			Completed:   result.Error == "",
		}
		if result.Error != "" {
			summary.Error = result.Error
		}
		summaries = append(summaries, summary)
	}

	taskArgs := map[string]any{}
	if report.Metadata != nil {
		for key, value := range report.Metadata {
			taskArgs[key] = value
		}
	}

	planParams := map[string]any{"model": report.ModelName}
	if report.Metadata != nil {
		for key, value := range report.Metadata {
			planParams[key] = value
		}
	}

	eval := EvalSpec{
		Created: startedAt.UTC().Format(time.RFC3339Nano),
		Task:    report.TaskName,
		Dataset: EvalDataset{
			Name:      report.TaskName,
			Samples:   &totalSamples,
			SampleIDs: sampleIDs,
		},
		Model: report.ModelName,
		Config: EvalConfig{
			Epochs:      &epochs,
			LogSamples:  &logSamples,
			LogRealtime: &logRealtime,
			LogImages:   &logImages,
		},
		TaskArgs: taskArgs,
	}

	plan := EvalPlan{
		Name: "plan",
		Steps: []EvalPlanStep{
			{Solver: "basic", Params: planParams},
		},
		Config: map[string]any{},
	}

	return EvalLog{
		Version: 2,
		Status:  "success",
		Eval:    eval,
		Plan:    plan,
		Results: results,
		Stats: EvalStats{
			StartedAt:   startedAt.UTC().Format(time.RFC3339Nano),
			CompletedAt: completedAt.UTC().Format(time.RFC3339Nano),
			ModelUsage:  map[string]ModelUsage{report.ModelName: usage},
		},
		Samples:     samples,
		Invalidated: false,
		Reductions:  nil,
	}
}

func WriteJSON(logDir string, log EvalLog) (string, error) {
	if logDir == "" {
		return "", fmt.Errorf("inspectlog: logDir is required")
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return "", err
	}

	filename := buildLogFileName(log, "json")
	path := filepath.Join(logDir, filename)
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(log); err != nil {
		return "", err
	}
	return path, nil
}

func WriteEval(logDir string, log EvalLog) (string, error) {
	if logDir == "" {
		return "", fmt.Errorf("inspectlog: logDir is required")
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return "", err
	}

	filename := buildLogFileName(log, "eval")
	path := filepath.Join(logDir, filename)
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	start := LogStart{Version: log.Version, Eval: log.Eval, Plan: log.Plan}
	if err := writeZipJSON(zipWriter, "_journal/start.json", start); err != nil {
		return "", err
	}

	header := log
	header.Samples = nil
	if err := writeZipJSON(zipWriter, "header.json", header); err != nil {
		return "", err
	}

	summaries := buildSummaries(log.Samples)
	if err := writeZipJSON(zipWriter, "summaries.json", summaries); err != nil {
		return "", err
	}

	for _, sample := range log.Samples {
		safeID := sanitizeName(sample.ID)
		if safeID == "" {
			safeID = "sample"
		}
		filename := fmt.Sprintf("samples/%s_epoch_%d.json", safeID, sample.Epoch)
		if err := writeZipJSON(zipWriter, filename, sample); err != nil {
			return "", err
		}
	}

	if len(log.Reductions) > 0 {
		if err := writeZipJSON(zipWriter, "reductions.json", log.Reductions); err != nil {
			return "", err
		}
	}

	return path, nil
}

func buildLogFileName(log EvalLog, ext string) string {
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	task := sanitizeName(log.Eval.Task)
	model := sanitizeName(log.Eval.Model)
	if task == "" {
		task = "task"
	}
	if model == "" {
		model = "model"
	}
	return fmt.Sprintf("%s_%s_%s.%s", timestamp, task, model, ext)
}

func writeZipJSON(writer *zip.Writer, name string, data any) error {
	entry, err := writer.Create(name)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(entry)
	return encoder.Encode(data)
}

func buildSummaries(samples []EvalSample) []EvalSampleSummary {
	summaries := make([]EvalSampleSummary, 0, len(samples))
	for _, sample := range samples {
		summary := EvalSampleSummary{
			ID:          sample.ID,
			Epoch:       sample.Epoch,
			Input:       sample.Input,
			Target:      sample.Target,
			Metadata:    sample.Metadata,
			Scores:      sample.Scores,
			ModelUsage:  sample.ModelUsage,
			TotalTime:   sample.TotalTime,
			WorkingTime: sample.WorkingTime,
			Completed:   sample.Error == nil,
		}
		if sample.Error != nil {
			summary.Error = sample.Error.Message
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

func mapStringStringToAny(input map[string]string) map[string]any {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func sanitizeName(input string) string {
	out := make([]rune, 0, len(input))
	for _, r := range input {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			out = append(out, r)
		}
	}
	return string(out)
}

func intPointer(value int) *int {
	return &value
}
