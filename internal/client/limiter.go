package client

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateLimiter struct {
	limiter *rate.Limiter
	mu      sync.Mutex
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Inf, 1),
	}
}

func (rl *RateLimiter) Wait(ctx context.Context) error {
	return rl.limiter.Wait(ctx)
}

func (rl *RateLimiter) Update(limit rate.Limit, burst int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.limiter.SetLimit(limit)
	rl.limiter.SetBurst(burst)
}

func (rl *RateLimiter) BlockFor(duration time.Duration) {
	rl.mu.Lock()
	rl.limiter.SetLimit(0)
	rl.mu.Unlock()

	time.AfterFunc(duration, func() {
		rl.mu.Lock()
		rl.limiter.SetLimit(rate.Inf)
		rl.mu.Unlock()
	})
}

func ParseRetryAfter(headers http.Header) time.Duration {
	retryAfter := headers.Get("Retry-After")
	if retryAfter == "" {
		return time.Minute // default
	}

	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		return time.Duration(seconds) * time.Second
	}

	if t, err := http.ParseTime(retryAfter); err == nil {
		return time.Until(t)
	}

	return time.Minute // fallback
}
