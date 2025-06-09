package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/denmor86/ya-gophermart/internal/logger"
	"github.com/denmor86/ya-gophermart/internal/models"
	"golang.org/x/time/rate"
)

var (
	ErrAccrualServiceUnavailable = errors.New("accrual service unavailable")
)

// AccrualService - представляет интерфейс для работы с сервисом начисления баллов лояльности пользователя
type AccrualService interface {
	GetOrderAccrual(ctx context.Context, orderNumber string) (float64, string, error)
}

type Accrual struct {
	AccrualAddr string
	Limiter     *rate.Limiter
	Mutex       sync.Mutex
}

// Создание сервиса
func NewAccrual(address string) AccrualService {
	return &Accrual{
		AccrualAddr: address,
		Limiter:     rate.NewLimiter(rate.Inf, 1),
	}
}

// GetOrderAccrual - основная функция для получения информации о заказе и обработки ответа
func (s *Accrual) GetOrderAccrual(ctx context.Context, orderNumber string) (float64, string, error) {
	// Ожидаем разрешения от лимитера
	if err := s.Limiter.Wait(ctx); err != nil {
		logger.Error("Wait limiter error", err)
		return 0, "", fmt.Errorf("limiter error: %w", err)
	}

	// Формирование запроса
	url := s.AccrualAddr + "/api/orders/" + orderNumber
	// Создаем запрос
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logger.Error("Error create request to accurual", err)
		return 0, "", fmt.Errorf("failed to create new request")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Request to accrual service failed", err)
		return 0, "", fmt.Errorf("request to accrual service failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.Error("Failed to close response body", closeErr)
		}
	}()

	// Обновление настроек лимитера из header
	s.UpdateRateLimit(resp.Header)

	// Обработка кода ответа
	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		logger.Warn("Too many requests to accrual service", orderNumber)

		// Получение заголовок Retry-After
		retryAfter := resp.Header.Get("Retry-After")

		var waitDuration time.Duration
		if retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				waitDuration = time.Duration(seconds) * time.Second
			} else if t, err := http.ParseTime(retryAfter); err == nil {
				waitDuration = time.Until(t)
			}
		} else {
			// Если заголовок отсутствует, ждем по умолчанию 1 минуту
			waitDuration = time.Minute
		}

		// Блокировка запросов на указанный период
		s.Mutex.Lock()
		s.Limiter.SetLimit(0)
		s.Mutex.Unlock()

		// Ставим ожидание времени для следующих запросов
		go func() {
			time.Sleep(waitDuration)
			s.Mutex.Lock()
			s.Limiter.SetLimit(rate.Inf)
			s.Mutex.Unlock()
		}()

		return 0, models.OrderStatusProcessing, nil

	case http.StatusNoContent:
		return 0, models.OrderStatusInvalid, nil

	case http.StatusOK:
		// Продолжаем обработку
	default:
		logger.Error("Accrual service returned an error", resp.StatusCode)
		return 0, "", ErrAccrualServiceUnavailable
	}

	// Декодируем JSON-ответ
	var result struct {
		Order   string  `json:"order"`
		Status  string  `json:"status"`
		Accrual float64 `json:"accrual,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Error("Error decode JSON response", err)
		return 0, "", fmt.Errorf("failed decode JSON response")
	}

	if result.Status != models.OrderStatusRegistered &&
		result.Status != models.OrderStatusProcessing &&
		result.Status != models.OrderStatusInvalid &&
		result.Status != models.OrderStatusProcessed {
		logger.Error("Undefined status request", result.Status)
		return 0, "", fmt.Errorf("undefined status request %s", result.Status)
	}

	return result.Accrual, result.Status, nil
}

// UpdateRateLimit обновляет rate limiter на основе заголовков ответа
func (s *Accrual) UpdateRateLimit(headers http.Header) {
	limitHeader := headers.Get("X-RateLimit-Limit")
	resetHeader := headers.Get("X-RateLimit-Reset")

	if limitHeader != "" && resetHeader != "" {
		limit, err := strconv.Atoi(limitHeader)
		if err != nil {
			return
		}

		resetUnix, err := strconv.ParseInt(resetHeader, 10, 64)
		if err != nil {
			return
		}

		resetTime := time.Unix(resetUnix, 0)
		window := time.Until(resetTime)

		if window > 0 && limit > 0 {
			newRate := rate.Limit(float64(limit) / window.Seconds())

			// Защищаем обновление лимитера мьютексом
			s.Mutex.Lock()
			s.Limiter.SetLimit(newRate)
			s.Limiter.SetBurst(limit)
			s.Mutex.Unlock()

			fmt.Printf("Updated rate limit: %d requests per %v\n", limit, window)
		}
	}
}
