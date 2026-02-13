package core

import (
	"context"
	"errors"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Evaluator runs a dataset through a solver and scorer.
type Evaluator struct {
	Dataset        Dataset
	Solver         Solver
	Scorer         Scorer
	Workers        int
	Progress       func(completed, total, inflight int)
	TotalSamples   int
	RateLimiter    RateLimiter
	SampleTimeout  time.Duration
	MaxTotalTokens int
}

// Run executes an evaluation and returns a report.
func (e *Evaluator) Run(ctx context.Context) (EvalReport, error) {
	if e.Dataset == nil || e.Solver == nil || e.Scorer == nil {
		return EvalReport{}, errors.New("evaluator: dataset, solver, and scorer are required")
	}

	workers := e.Workers
	if workers <= 0 {
		workers = 1
	}

	started := time.Now()

	runCtx, runCancel := context.WithCancel(ctx)
	defer runCancel()

	sampleCh, errCh := e.Dataset.Samples(runCtx)

	resultsCh := make(chan EvalResult, workers)
	var wg sync.WaitGroup
	var completed int64
	var inflight int64

	worker := func() {
		defer wg.Done()
		for sample := range sampleCh {
			select {
			case <-runCtx.Done():
				return
			default:
			}

			if e.RateLimiter != nil {
				if err := e.RateLimiter.Wait(runCtx); err != nil {
					return
				}
			}

			currentInflight := atomic.AddInt64(&inflight, 1)
			if e.Progress != nil {
				e.Progress(int(atomic.LoadInt64(&completed)), e.TotalSamples, int(currentInflight))
			}

			sampleCtx := runCtx
			var sampleCancel context.CancelFunc
			if e.SampleTimeout > 0 {
				sampleCtx, sampleCancel = context.WithTimeout(runCtx, e.SampleTimeout)
			}
			result := evaluateSample(sampleCtx, e.Solver, e.Scorer, sample)
			timedOut := sampleCtx.Err() != nil && sampleCancel != nil
			if sampleCancel != nil {
				sampleCancel()
			}
			if result.Error == "" && timedOut {
				result.Error = "sample timeout"
			}
			select {
			case resultsCh <- result:
				currentInflight = atomic.AddInt64(&inflight, -1)
				if e.Progress != nil {
					e.Progress(int(atomic.LoadInt64(&completed)), e.TotalSamples, int(currentInflight))
				}
			case <-runCtx.Done():
				currentInflight = atomic.AddInt64(&inflight, -1)
				if e.Progress != nil {
					e.Progress(int(atomic.LoadInt64(&completed)), e.TotalSamples, int(currentInflight))
				}
				return
			}
		}
	}

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go worker()
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	var results []EvalResult
	var datasetErr error
	var cumulativeTokens int
	for {
		select {
		case <-ctx.Done():
			return EvalReport{}, ctx.Err()
		case err, ok := <-errCh:
			if ok && err != nil && datasetErr == nil {
				datasetErr = err
			}
		case result, ok := <-resultsCh:
			if !ok {
				if datasetErr != nil {
					return EvalReport{}, datasetErr
				}
				report := EvalReport{
					TaskName:   e.Dataset.Name(),
					ModelName:  e.Solver.Name(),
					ScorerName: e.Scorer.Name(),
					Metrics:    CalculateMetrics(results),
					Results:    results,
					StartedAt:  started,
					FinishedAt: time.Now(),
				}
				return report, nil
			}
			results = append(results, result)
			if e.Progress != nil {
				currentCompleted := atomic.AddInt64(&completed, 1)
				e.Progress(int(currentCompleted), e.TotalSamples, int(atomic.LoadInt64(&inflight)))
			}
			if e.MaxTotalTokens > 0 {
				cumulativeTokens += result.Response.TokenUsage.TotalTokens
				if cumulativeTokens >= e.MaxTotalTokens {
					runCancel()
				}
			}
		}
	}
}

func evaluateSample(ctx context.Context, solver Solver, scorer Scorer, sample Sample) EvalResult {
	start := time.Now()
	result := EvalResult{Sample: sample}

	response, err := solver.Solve(ctx, sample)
	if err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}

	score, err := scorer.Score(ctx, sample, response)
	if err != nil {
		result.Error = err.Error()
	}
	result.Response = response
	result.Score = score
	result.Duration = time.Since(start)
	return result
}

func CalculateMetrics(results []EvalResult) Metrics {
	if len(results) == 0 {
		return Metrics{}
	}

	scores := make([]float64, 0, len(results))
	latencies := make([]time.Duration, 0, len(results))
	var passed int
	var totalTokens TokenUsage

	for _, result := range results {
		scores = append(scores, result.Score.Value)
		latencies = append(latencies, result.Response.Latency)
		if result.Score.Passed {
			passed++
		}
		totalTokens.PromptTokens += result.Response.TokenUsage.PromptTokens
		totalTokens.CompletionTokens += result.Response.TokenUsage.CompletionTokens
		totalTokens.TotalTokens += result.Response.TokenUsage.TotalTokens
	}

	return Metrics{
		TotalSamples: len(results),
		SuccessRate:  float64(passed) / float64(len(results)),
		AverageScore: average(scores),
		MedianScore:  percentile(scores, 0.50),
		P50Score:     percentile(scores, 0.50),
		P95Score:     percentile(scores, 0.95),
		P99Score:     percentile(scores, 0.99),
		TokenUsage:   totalTokens,
		AvgLatency:   averageDuration(latencies),
		P50Latency:   percentileDuration(latencies, 0.50),
		P95Latency:   percentileDuration(latencies, 0.95),
		P99Latency:   percentileDuration(latencies, 0.99),
	}
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	copied := make([]float64, len(values))
	copy(copied, values)
	sort.Float64s(copied)

	if p <= 0 {
		return copied[0]
	}
	if p >= 1 {
		return copied[len(copied)-1]
	}

	index := p * float64(len(copied)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	if lower == upper {
		return copied[lower]
	}
	weight := index - float64(lower)
	return copied[lower]*(1-weight) + copied[upper]*weight
}

func averageDuration(values []time.Duration) time.Duration {
	if len(values) == 0 {
		return 0
	}
	var sum time.Duration
	for _, v := range values {
		sum += v
	}
	return time.Duration(int64(sum) / int64(len(values)))
}

func percentileDuration(values []time.Duration, p float64) time.Duration {
	if len(values) == 0 {
		return 0
	}
	copied := make([]time.Duration, len(values))
	copy(copied, values)
	sort.Slice(copied, func(i, j int) bool { return copied[i] < copied[j] })

	if p <= 0 {
		return copied[0]
	}
	if p >= 1 {
		return copied[len(copied)-1]
	}

	index := p * float64(len(copied)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	if lower == upper {
		return copied[lower]
	}
	weight := index - float64(lower)
	lowerVal := float64(copied[lower])
	upperVal := float64(copied[upper])
	return time.Duration(lowerVal*(1-weight) + upperVal*weight)
}
