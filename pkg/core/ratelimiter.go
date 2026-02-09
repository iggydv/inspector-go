package core

import (
	"context"
	"errors"
	"math"
	"time"
)

type RateLimiter interface {
	Wait(ctx context.Context) error
}

type tokenBucket struct {
	tokens chan struct{}
	stop   chan struct{}
}

func NewRateLimiter(rps float64, burst int) (RateLimiter, func(), error) {
	if rps <= 0 {
		return nil, func() {}, errors.New("rate limiter: rps must be > 0")
	}
	if burst <= 0 {
		burst = 1
	}

	interval := time.Duration(math.Round(float64(time.Second) / rps))
	if interval <= 0 {
		interval = time.Nanosecond
	}

	tb := &tokenBucket{
		tokens: make(chan struct{}, burst),
		stop:   make(chan struct{}),
	}

	for i := 0; i < burst; i++ {
		tb.tokens <- struct{}{}
	}

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-tb.stop:
				return
			case <-ticker.C:
				select {
				case tb.tokens <- struct{}{}:
				default:
				}
			}
		}
	}()

	stop := func() {
		close(tb.stop)
	}

	return tb, stop, nil
}

func (t *tokenBucket) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.tokens:
		return nil
	}
}
