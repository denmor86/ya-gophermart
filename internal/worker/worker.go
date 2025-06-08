package worker

import (
	"context"
	"sync"
	"time"

	"github.com/denmor86/ya-gophermart/internal/logger"
	"github.com/denmor86/ya-gophermart/internal/services"
	"github.com/sony/gobreaker"
)

func InitCircuitBreaker() *gobreaker.CircuitBreaker {
	return gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:    "accrual-service",
		Timeout: 30 * time.Second, // через 30 сек пробуем подключиться
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// 5 попыток достучатся до сервиса
			return counts.ConsecutiveFailures >= 5
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			logger.Info("Circuit Breaker '%s': %s → %s", name, from, to)
		},
	})
}

// OrderWorker - основной воркер для обработки заказов
type OrderWorker struct {
	Orders       services.OrdersService
	Breaker      *gobreaker.CircuitBreaker
	WaitGroup    sync.WaitGroup
	QuitChan     chan struct{}
	BatchSize    int
	PollInterval time.Duration
}

// NewOrderWorker - конструктор обработчика системы расчёта вознаграждений
func NewOrderWorker(orders services.OrdersService) *OrderWorker {
	return &OrderWorker{
		Orders:       orders,
		Breaker:      InitCircuitBreaker(),
		QuitChan:     make(chan struct{}),
		BatchSize:    10,
		PollInterval: 5 * time.Second,
	}
}

// Start - запускает воркер в фоне
func (w *OrderWorker) Start(ctx context.Context) {
	w.WaitGroup.Add(1)
	go w.Run(ctx)
}

// Stop - корректно останавливает воркер
func (w *OrderWorker) Stop() {
	close(w.QuitChan)
	w.WaitGroup.Wait()
}

// run - основная рабочая логика
func (w *OrderWorker) Run(ctx context.Context) {
	defer w.WaitGroup.Done()

	ticker := time.NewTicker(w.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.QuitChan:
			logger.Info("OrderWorker signal stop")
			return
		case <-ticker.C:
			w.ProcessOrder(ctx)
		}
	}
}

// ProcessOrder - обработка пачки заказов
func (w *OrderWorker) ProcessOrder(ctx context.Context) {
	if w.Breaker.State() == gobreaker.StateOpen {
		logger.Warn("%s unavailable. Waiting...", w.Breaker.Name())
		return
	}

	orderNumbers, err := w.Orders.GetProcessingOrders(ctx, w.BatchSize)

	if err != nil {
		logger.Error("error get orders for processing", err)
		return
	}

	for _, orderNumber := range orderNumbers {
		_, err := w.Breaker.Execute(func() (interface{}, error) {
			return nil, w.Orders.ProcessOrder(ctx, orderNumber)
		})

		if err != nil {
			logger.Error("Error order processing", err)
		}
	}
}
