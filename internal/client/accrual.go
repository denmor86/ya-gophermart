package client

import (
	"context"
	"errors"
	"net/http"
	"time"
)

type OrderResponse struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
}

type AccrualService interface {
	GetOrderAccrual(ctx context.Context, orderNumber string) (float64, string, error)
}

type RateLimitConfig struct {
	Limit int
	Reset int64
}

var (
	ErrServiceUnavailable = errors.New("accrual service unavailable")
	ErrOrderNotRegistered = errors.New("order not registered")
)

type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return "rate limit exceeded"
}

func NewRateLimitError(headers http.Header) *RateLimitError {
	return &RateLimitError{
		RetryAfter: ParseRetryAfter(headers),
	}
}
