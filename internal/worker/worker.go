package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/denmor86/ya-gophermart/internal/config"
	"github.com/denmor86/ya-gophermart/internal/logger"
	"github.com/denmor86/ya-gophermart/internal/services"
	"github.com/sony/gobreaker"
	"go.uber.org/zap"
)

type OrderWorker struct {
	Orders    services.OrdersService
	Breaker   *gobreaker.CircuitBreaker
	WaitGroup sync.WaitGroup
	QuitChan  chan struct{}
	config    config.AccrualConfig
}

func NewOrderWorker(orders services.OrdersService, config config.AccrualConfig) *OrderWorker {
	return &OrderWorker{
		Orders: orders,
		Breaker: gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "accrual-service",
			Timeout: 30 * time.Second,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= 5
			},
			OnStateChange: func(name string, from, to gobreaker.State) {
				logger.Info("Circuit Breaker '%s': %s → %s", name, from, to)
			},
		}),
		QuitChan: make(chan struct{}),
		config:   config,
	}
}

func (w *OrderWorker) Start(ctx context.Context) {
	w.WaitGroup.Add(1)
	go w.Run(ctx)
}

func (w *OrderWorker) Stop() {
	close(w.QuitChan)
	w.WaitGroup.Wait()
}

func (w *OrderWorker) Run(ctx context.Context) {
	defer w.WaitGroup.Done()

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.QuitChan:
			logger.Info("OrderWorker stopped by quit signal")
			return
		case <-ctx.Done():
			logger.Info("OrderWorker stopped by context cancellation")
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}
func (w *OrderWorker) processBatch(ctx context.Context) {
	// Используем Circuit Breaker для всей операции
	_, err := w.Breaker.Execute(func() (interface{}, error) {
		// Получаем заказы для обработки
		orders, err := w.Orders.ClaimOrdersForProcessing(ctx, w.config.BatchSize)
		if err != nil {
			logger.Error("Failed to claim orders:", zap.Error(err))
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
			return nil, fmt.Errorf("failed to claim orders: %w", err)
		}

		// Последовательная обработка заказов
		var lastErr error
		for _, orderNum := range orders {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				// Обрабатываем заказ с индивидуальным таймаутом
				processCtx, cancel := context.WithTimeout(ctx, w.config.ProcessingTimeout)
				defer cancel()

				processErr := w.Orders.ProcessOrder(processCtx, orderNum)
				if processErr != nil {
					logger.Error("Failed to process order", orderNum, "Error:", zap.Error(processErr))
					lastErr = processErr
					// Продолжаем обработку следующих заказов
				}
			}
		}

		return nil, lastErr
	})

	if err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("Order batch processing failed:", zap.Error(err))
	}
}
