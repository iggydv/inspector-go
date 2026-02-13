package inspectlog

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"strconv"
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
	Invalidated bool                  `json:"invalidated"`
	Samples     []EvalSample          `json:"samples,omitempty"`
	Reductions  []EvalSampleReduction `json:"reductions,omitempty"`
}

type EvalError struct {
	Message       string `json:"message"`
	Traceback     string `json:"traceback"`
	TracebackANSI string `json:"traceback_ansi"`
}

type EvalSpec struct {
	Created             string         `json:"created"`
	Task                string         `json:"task"`
	Dataset             EvalDataset    `json:"dataset"`
	Model               string         `json:"model"`
	Config              EvalConfig     `json:"config"`
	TaskArgs            map[string]any `json:"task_args"`
	TaskArgsPassed      map[string]any `json:"task_args_passed"`
	ModelArgs           map[string]any `json:"model_args"`
	ModelGenerateConfig map[string]any `json:"model_generate_config"`
	Packages            map[string]any `json:"packages"`
	TaskAttribs         map[string]any `json:"task_attribs"`
	Scorers             []any          `json:"scorers,omitempty"`

	EvalID           string `json:"eval_id"`
	RunID            string `json:"run_id"`
	TaskID           string `json:"task_id"`
	TaskVersion      int    `json:"task_version"`
	TaskFile         string `json:"task_file,omitempty"`
	TaskRegistryName string `json:"task_registry_name,omitempty"`
	TaskDisplayName  string `json:"task_display_name,omitempty"`
	Revision         any    `json:"revision,omitempty"`
}

type EvalDataset struct {
	Name      string `json:"name,omitempty"`
	Location  string `json:"location,omitempty"`
	Samples   int    `json:"samples"`
	SampleIDs []int  `json:"sample_ids,omitempty"`
	Shuffled  bool   `json:"shuffled"`
}

type EvalConfig struct {
	Epochs         int      `json:"epochs"`
	EpochsReducer  []string `json:"epochs_reducer,omitempty"`
	FailOnError    bool     `json:"fail_on_error"`
	ContinueOnFail bool     `json:"continue_on_fail"`
	SandboxCleanup bool     `json:"sandbox_cleanup"`
	LogSamples     bool     `json:"log_samples"`
	LogRealtime    bool     `json:"log_realtime"`
	LogImages      bool     `json:"log_images"`
	ScoreDisplay   bool     `json:"score_display"`
}

type EvalPlan struct {
	Name   string         `json:"name,omitempty"`
	Steps  []EvalPlanStep `json:"steps,omitempty"`
	Finish *EvalPlanStep  `json:"finish,omitempty"`
	Config map[string]any `json:"config"`
}

type EvalPlanStep struct {
	Solver       string         `json:"solver"`
	Params       map[string]any `json:"params"`
	ParamsPassed map[string]any `json:"params_passed"`
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
	Params          map[string]any        `json:"params"`
	Metrics         map[string]EvalMetric `json:"metrics,omitempty"`
	Metadata        map[string]any        `json:"metadata,omitempty"`
}

type EvalMetric struct {
	Name     string         `json:"name"`
	Value    float64        `json:"value"`
	Params   map[string]any `json:"params"`
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
	ID           int                   `json:"id"`
	Epoch        int                   `json:"epoch"`
	Input        string                `json:"input"`
	Target       string                `json:"target"`
	Messages     []any                 `json:"messages"`
	Output       ModelOutput           `json:"output"`
	Scores       map[string]Score      `json:"scores"`
	Metadata     map[string]any        `json:"metadata"`
	Store        map[string]any        `json:"store"`
	Events       []any                 `json:"events"`
	Attachments  map[string]any        `json:"attachments"`
	ErrorRetries []any                 `json:"error_retries"`
	ModelUsage   map[string]ModelUsage `json:"model_usage"`
	StartedAt    string                `json:"started_at"`
	CompletedAt  string                `json:"completed_at"`
	TotalTime    float64               `json:"total_time"`
	WorkingTime  float64               `json:"working_time"`
	UUID         string                `json:"uuid"`
	Error        *EvalError            `json:"error,omitempty"`
}

type ModelOutput struct {
	Model      string      `json:"model"`
	Choices    []any       `json:"choices"`
	Completion string      `json:"completion"`
	Usage      *ModelUsage `json:"usage"`
	Time       *float64    `json:"time"`
}

type Score struct {
	Value       any    `json:"value"`
	Answer      string `json:"answer"`
	Explanation string `json:"explanation"`
	History     []any  `json:"history"`
}

type EvalSampleSummary struct {
	ID           int                   `json:"id"`
	Epoch        int                   `json:"epoch"`
	Input        string                `json:"input"`
	Target       string                `json:"target"`
	Metadata     map[string]any        `json:"metadata"`
	Scores       map[string]Score      `json:"scores"`
	ModelUsage   map[string]ModelUsage `json:"model_usage"`
	StartedAt    string                `json:"started_at"`
	CompletedAt  string                `json:"completed_at"`
	TotalTime    float64               `json:"total_time"`
	WorkingTime  float64               `json:"working_time"`
	MessageCount int                   `json:"message_count"`
	Retries      int                   `json:"retries"`
	UUID         string                `json:"uuid"`
	Error        string                `json:"error,omitempty"`
	Completed    bool                  `json:"completed"`
}

type EvalSampleReduction struct {
	Scorer  string        `json:"scorer"`
	Reducer string        `json:"reducer,omitempty"`
	Samples []SampleScore `json:"samples"`
}

type SampleScore struct {
	SampleID    string `json:"sample_id"`
	Value       any    `json:"value"`
	Answer      string `json:"answer"`
	Explanation string `json:"explanation"`
	History     []any  `json:"history"`
}

type LogStart struct {
	Version int      `json:"version"`
	Eval    EvalSpec `json:"eval"`
	Plan    EvalPlan `json:"plan"`
}

func FromReport(report core.EvalReport) EvalLog {
	const timeLayout = "2006-01-02T15:04:05-07:00"

	scoreName := report.ScorerName
	if scoreName == "" {
		scoreName = "score"
	}

	modelName := report.ModelName

	startedAt := report.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	completedAt := report.FinishedAt
	if completedAt.IsZero() {
		completedAt = time.Now().UTC()
	}

	sampleIDs := make([]int, 0, len(report.Results))
	for idx, result := range report.Results {
		if id, err := strconv.Atoi(result.Sample.ID); err == nil {
			sampleIDs = append(sampleIDs, id)
		} else {
			sampleIDs = append(sampleIDs, idx+1)
		}
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
	// Keep reported token usage; do not override.

	metrics := map[string]EvalMetric{
		"success_rate":  {Name: "success_rate", Value: report.Metrics.SuccessRate, Params: map[string]any{}},
		"average_score": {Name: "average_score", Value: report.Metrics.AverageScore, Params: map[string]any{}},
		"median_score":  {Name: "median_score", Value: report.Metrics.MedianScore, Params: map[string]any{}},
		"p95_score":     {Name: "p95_score", Value: report.Metrics.P95Score, Params: map[string]any{}},
		"p99_score":     {Name: "p99_score", Value: report.Metrics.P99Score, Params: map[string]any{}},
	}

	score := EvalScore{
		Name:            scoreName,
		Scorer:          scoreName,
		ScoredSamples:   &totalSamples,
		UnscoredSamples: intPointer(totalSamples - completedSamples),
		Params:          map[string]any{},
		Metrics:         metrics,
	}

	results := &EvalResults{
		TotalSamples:     totalSamples,
		CompletedSamples: completedSamples,
		Scores:           []EvalScore{score},
	}

	// Build reductions
	reductions := []EvalSampleReduction{
		{
			Scorer:  scoreName,
			Samples: []SampleScore{},
		},
	}

	// Add each sample's score to reductions
	for idx, result := range report.Results {
		id := fmt.Sprintf("%d", idx+1)
		reductionAnswer := result.Response.Content

		reductionValue := any(result.Score.Value)
		if result.Score.Passed {
			reductionValue = "C"
		} else {
			reductionValue = "I"
		}
		reductions[0].Samples = append(reductions[0].Samples, SampleScore{
			SampleID:    id,
			Value:       reductionValue,
			Answer:      reductionAnswer,
			Explanation: reductionAnswer,
			History:     []any{},
		})
	}

	samples := make([]EvalSample, 0, len(report.Results))
	summaries := make([]EvalSampleSummary, 0, len(report.Results))

	for idx, result := range report.Results {
		id := result.Sample.ID
		if id == "" {
			id = fmt.Sprintf("%d", idx+1)
		}

		modelUsage := map[string]ModelUsage{}
		if modelName != "" {
			modelUsage[modelName] = usage
		}

		outputUsage := usage
		outputTime := result.Response.Latency.Seconds()
		scoreValue := any(result.Score.Value)
		if result.Score.Passed {
			scoreValue = "C"
		} else {
			scoreValue = "I"
		}
		sampleScore := map[string]Score{
			scoreName: {
				Value:       scoreValue,
				Answer:      result.Response.Content,
				Explanation: result.Response.Content,
				History:     []any{},
			},
		}

		var sampleError *EvalError
		if result.Error != "" {
			sampleError = &EvalError{
				Message:       result.Error,
				Traceback:     "",
				TracebackANSI: "",
			}
		}

		messageUserID := generateID()
		messageAssistantID := generateID()
		assistantContent := result.Response.Content

		sample := EvalSample{
			ID:     idx + 1,
			Epoch:  1,
			Input:  result.Sample.Input,
			Target: result.Sample.Expected,
			Messages: []any{
				map[string]any{
					"id":      messageUserID,
					"content": result.Sample.Input,
					"source":  "input",
					"role":    "user",
				},
				map[string]any{
					"id":      messageAssistantID,
					"content": assistantContent,
					"source":  "generate",
					"role":    "assistant",
					"model":   modelName,
				},
			},
			Output: ModelOutput{
				Model: modelName,
				Choices: []any{
					map[string]any{
						"message": map[string]any{
							"id":      messageAssistantID,
							"content": assistantContent,
							"source":  "generate",
							"role":    "assistant",
							"model":   modelName,
						},
						"stop_reason": "stop",
					},
				},
				Completion: assistantContent,
				Usage:      &outputUsage,
				Time:       &outputTime,
			},
			Scores:       sampleScore,
			Metadata:     map[string]any{},
			Store:        map[string]any{},
			Events:       []any{},
			Attachments:  map[string]any{},
			ErrorRetries: []any{},
			ModelUsage:   modelUsage,
			StartedAt:    startedAt.UTC().Format(timeLayout),
			CompletedAt:  completedAt.UTC().Format(timeLayout),
			TotalTime:    result.Duration.Seconds(),
			WorkingTime:  result.Duration.Seconds(),
			UUID:         generateID(),
			Error:        sampleError,
		}
		samples = append(samples, sample)

		summary := EvalSampleSummary{
			ID:           idx + 1,
			Epoch:        1,
			Input:        result.Sample.Input,
			Target:       result.Sample.Expected,
			Metadata:     mapStringStringToAny(result.Sample.Metadata),
			Scores:       sampleScore,
			ModelUsage:   modelUsage,
			StartedAt:    startedAt.UTC().Format(timeLayout),
			CompletedAt:  completedAt.UTC().Format(timeLayout),
			TotalTime:    result.Duration.Seconds(),
			WorkingTime:  result.Duration.Seconds(),
			MessageCount: 3,
			Retries:      0,
			UUID:         generateID(),
			Completed:    result.Error == "",
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
	planParams := map[string]any{}
	if report.Metadata != nil {
		for key, value := range report.Metadata {
			planParams[key] = value
		}
	}

	// Build scorers array
	scorers := []any{
		map[string]any{
			"name":    scoreName,
			"options": map[string]any{},
			"metrics": []any{
				map[string]any{"name": "inspect_ai/accuracy", "options": map[string]any{}},
				map[string]any{"name": "inspect_ai/stderr", "options": map[string]any{}},
			},
			"metadata": map[string]any{},
		},
	}

	taskName := report.TaskName
	eval := EvalSpec{
		Created: startedAt.UTC().Format(timeLayout),
		Task:    taskName,
		Dataset: EvalDataset{
			Name:      report.TaskName,
			Samples:   totalSamples,
			SampleIDs: sampleIDs,
			Shuffled:  false,
		},
		Model: modelName,
		Config: EvalConfig{
			Epochs:         1,
			EpochsReducer:  []string{"mean"},
			FailOnError:    true,
			ContinueOnFail: false,
			SandboxCleanup: true,
			LogSamples:     true,
			LogRealtime:    true,
			LogImages:      true,
			ScoreDisplay:   true,
		},
		TaskArgs:            taskArgs,
		TaskArgsPassed:      map[string]any{},
		ModelArgs:           map[string]any{},
		ModelGenerateConfig: map[string]any{},
		Packages:            map[string]any{},
		TaskAttribs:         map[string]any{},
		Scorers:             scorers,
		EvalID:              generateID(),
		RunID:               generateID(),
		TaskID:              generateID(),
		TaskVersion:         0,
		TaskFile:            "",
		TaskRegistryName:    taskName,
		TaskDisplayName:     taskName,
		Revision:            nil,
	}

	solverName := ""
	if val, ok := taskArgs["solver"].(string); ok {
		solverName = val
	}

	plan := EvalPlan{
		Name: "plan",
		Steps: []EvalPlanStep{
			{
				Solver:       solverName,
				Params:       planParams,
				ParamsPassed: map[string]any{},
			},
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
			StartedAt:   startedAt.UTC().Format(timeLayout),
			CompletedAt: completedAt.UTC().Format(timeLayout),
			ModelUsage:  map[string]ModelUsage{modelName: usage},
		},
		Samples:     samples,
		Invalidated: false,
		Reductions:  reductions,
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

	// Write one summary file per sample (Python-style)
	for idx, summary := range summaries {
		summaryFile := fmt.Sprintf("_journal/summaries/%d.json", idx+1)
		if err := writeZipJSON(zipWriter, summaryFile, summary); err != nil {
			return "", err
		}
	}

	for _, sample := range log.Samples {
		filename := fmt.Sprintf("samples/%d_epoch_%d.json", sample.ID, sample.Epoch)
		if err := writeZipJSON(zipWriter, filename, sample); err != nil {
			return "", err
		}
	}

	// Always write reductions
	if err := writeZipJSON(zipWriter, "reductions.json", log.Reductions); err != nil {
		return "", err
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
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return err
	}

	payload := buf.Bytes()
	size := uint64(len(payload))
	header := &zip.FileHeader{
		Name:               name,
		Method:             zip.Store,
		Flags:              0,
		UncompressedSize64: size,
		CompressedSize64:   size,
		UncompressedSize:   uint32(size),
		CompressedSize:     uint32(size),
		CRC32:              crc32.ChecksumIEEE(payload),
	}
	header.SetModTime(time.Unix(0, 0))

	header.Flags &^= 0x8 // ensure no data descriptor
	entry, err := writer.CreateRaw(header)
	if err != nil {
		return err
	}
	if _, err := entry.Write(payload); err != nil {
		return err
	}
	return nil
}

func buildSummaries(samples []EvalSample) []EvalSampleSummary {
	summaries := make([]EvalSampleSummary, 0, len(samples))
	for idx, sample := range samples {
		messageCount := len(sample.Messages)
		retries := len(sample.ErrorRetries)
		summary := EvalSampleSummary{
			ID:           idx + 1,
			Epoch:        sample.Epoch,
			Input:        sample.Input,
			Target:       sample.Target,
			Metadata:     sample.Metadata,
			Scores:       sample.Scores,
			ModelUsage:   sample.ModelUsage,
			StartedAt:    sample.StartedAt,
			CompletedAt:  sample.CompletedAt,
			TotalTime:    sample.TotalTime,
			WorkingTime:  sample.WorkingTime,
			MessageCount: messageCount,
			Retries:      retries,
			UUID:         sample.UUID,
			Completed:    sample.Error == nil,
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

func generateID() string {
	// Generate 16 random bytes
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	// Encode to base58-like string (using base64 URL encoding without padding)
	return base64.RawURLEncoding.EncodeToString(b)
}

func ReadJSON(path string) (EvalLog, error) {
	var log EvalLog
	f, err := os.Open(path)
	if err != nil {
		return EvalLog{}, err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&log); err != nil {
		return EvalLog{}, err
	}
	return log, nil
}

func ReadEval(path string) (EvalLog, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return EvalLog{}, err
	}
	defer r.Close()

	var header EvalLog
	for _, f := range r.File {
		if f.Name == "header.json" {
			rc, err := f.Open()
			if err != nil {
				return EvalLog{}, err
			}
			err = json.NewDecoder(rc).Decode(&header)
			rc.Close()
			if err != nil {
				return EvalLog{}, err
			}
			break
		}
	}

	if header.Samples == nil {
		header.Samples = []EvalSample{}
	}
	for _, f := range r.File {
		if dir := filepath.Dir(f.Name); dir != "samples" || filepath.Ext(f.Name) != ".json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return EvalLog{}, err
		}
		var sample EvalSample
		decodeErr := json.NewDecoder(rc).Decode(&sample)
		rc.Close()
		if decodeErr != nil {
			return EvalLog{}, decodeErr
		}
		header.Samples = append(header.Samples, sample)
	}
	return header, nil
}

func FailedSamples(log EvalLog) []core.Sample {
	var out []core.Sample
	for _, s := range log.Samples {
		if s.Error != nil || (s.Output.Completion == "" && s.Error == nil) {
			meta := map[string]string{}
			if s.Metadata != nil {
				for k, v := range s.Metadata {
					if str, ok := v.(string); ok {
						meta[k] = str
					}
				}
			}
			out = append(out, core.Sample{
				ID:       fmt.Sprintf("%d", s.ID),
				Input:    s.Input,
				Expected: s.Target,
				Metadata: meta,
			})
		}
	}
	return out
}

func LogToReport(log EvalLog) core.EvalReport {
	scoreName := log.Eval.Model
	if len(log.Eval.Scorers) > 0 {
		if m, ok := log.Eval.Scorers[0].(map[string]any); ok {
			if n, _ := m["name"].(string); n != "" {
				scoreName = n
			}
		}
	}
	results := make([]core.EvalResult, 0, len(log.Samples))
	var totalUsage core.TokenUsage
	for _, s := range log.Samples {
		usage := core.TokenUsage{}
		if s.ModelUsage != nil {
			for _, u := range s.ModelUsage {
				usage.PromptTokens += u.InputTokens
				usage.CompletionTokens += u.OutputTokens
				usage.TotalTokens += u.TotalTokens
				break
			}
		}
		totalUsage.PromptTokens += usage.PromptTokens
		totalUsage.CompletionTokens += usage.CompletionTokens
		totalUsage.TotalTokens += usage.TotalTokens
		scoreVal := 0.0
		passed := false
		if s.Scores != nil {
			for _, sc := range s.Scores {
				if v, ok := sc.Value.(float64); ok {
					scoreVal = v
				}
				if v, ok := sc.Value.(string); ok && (v == "C" || v == "c") {
					passed = true
					scoreVal = 1.0
				}
				break
			}
		}
		errStr := ""
		if s.Error != nil {
			errStr = s.Error.Message
		}
		dur := time.Duration(0)
		if s.TotalTime > 0 {
			dur = time.Duration(s.TotalTime * float64(time.Second))
		}
		meta := map[string]string{}
		if s.Metadata != nil {
			for k, v := range s.Metadata {
				if str, ok := v.(string); ok {
					meta[k] = str
				}
			}
		}
		results = append(results, core.EvalResult{
			Sample: core.Sample{
				ID:       fmt.Sprintf("%d", s.ID),
				Input:    s.Input,
				Expected: s.Target,
				Metadata: meta,
			},
			Response: core.Response{
				Content:    s.Output.Completion,
				TokenUsage: usage,
				Latency:    dur,
			},
			Score: core.Score{
				Value:  scoreVal,
				Max:    1,
				Passed: passed,
			},
			Error:    errStr,
			Duration: dur,
		})
	}
	metrics := core.Metrics{}
	if len(results) > 0 {
		metrics = core.CalculateMetrics(results)
	}
	const timeLayout = "2006-01-02T15:04:05-07:00"
	var startedAt, finishedAt time.Time
	if t, err := time.Parse(timeLayout, log.Stats.StartedAt); err == nil {
		startedAt = t
	}
	if t, err := time.Parse(timeLayout, log.Stats.CompletedAt); err == nil {
		finishedAt = t
	}
	return core.EvalReport{
		TaskName:   log.Eval.Task,
		ModelName:  log.Eval.Model,
		ScorerName: scoreName,
		Metrics:    metrics,
		Results:    results,
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
	}
}
