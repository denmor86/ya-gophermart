package services

import (
	"context"
	"fmt"
	"net/http"

	"github.com/denmor86/ya-gophermart/internal/client"
	"github.com/denmor86/ya-gophermart/internal/logger"
	"github.com/denmor86/ya-gophermart/internal/models"
)

type AccrualService struct {
	Client  *client.Client
	Limiter *client.RateLimiter
}

func NewAccrualService(baseURL string) client.AccrualService {
	return &AccrualService{
		Client:  client.NewClient(baseURL, &http.Client{}),
		Limiter: client.NewRateLimiter(),
	}
}

func (s *AccrualService) GetOrderAccrual(ctx context.Context, orderNumber string) (float64, string, error) {
	if err := s.Limiter.Wait(ctx); err != nil {
		return 0, "", err
	}

	resp, err := s.Client.GetOrder(ctx, orderNumber)
	if err != nil {
		// проверка большого количеста запросов
		if rateLimitErr, ok := err.(*client.RateLimitError); ok {
			logger.Warn("Too many requests to accrual service:", orderNumber)
			s.Limiter.BlockFor(rateLimitErr.RetryAfter)
			return 0, models.OrderStatusProcessing, nil
		}
		return 0, models.OrderStatusInvalid, err
	}
	// проверяем возможные статусы
	if resp.Status != models.OrderStatusRegistered &&
		resp.Status != models.OrderStatusProcessing &&
		resp.Status != models.OrderStatusInvalid &&
		resp.Status != models.OrderStatusProcessed {
		logger.Error("Undefined status request:", resp.Status)
		return 0, "", fmt.Errorf("undefined status request %s", resp.Status)
	}
	return resp.Accrual, resp.Status, nil
}
