package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/denmor86/ya-gophermart/internal/config"
	"github.com/denmor86/ya-gophermart/internal/logger"
	"github.com/denmor86/ya-gophermart/internal/models"
	"golang.org/x/time/rate"
)

type testCase struct {
	TestName        string
	OrderNumber     string
	StatusCode      int
	Response        string
	ExpectedAccrual float64
	ExpectedStatus  string
	ExpectedError   error
}

func TestNewAccrualService(t *testing.T) {

	config := config.DefaultConfig()
	if err := logger.Initialize(config.Server.LogLevel); err != nil {
		logger.Panic(err)
	}
	defer logger.Sync()

	service := NewAccrual(config.Accrual.AccrualAddr)

	baseService, ok := service.(*Accrual)
	if !ok {
		t.Fatalf("Expected *Accrual, got '%T'", service)
	}
	// проверка адреса сервиса
	if baseService.AccrualAddr != config.Accrual.AccrualAddr {
		t.Errorf("Expected accrual address: '%v', got: '%v'", baseService.AccrualAddr, config.Accrual.AccrualAddr)
	}
	// Проверка лимитера
	if baseService.Limiter == nil {
		t.Error("Expected limiter to be initialized")
	} else if baseService.Limiter.Limit() != rate.Inf {
		t.Errorf("Expected limiter limit to be rate.Inf, got: '%v'", baseService.Limiter.Limit())
	}
}

func TestGetOrderAccrual(t *testing.T) {
	testCases := []testCase{
		{
			TestName:        "Success. Order processed #1",
			OrderNumber:     "123456",
			StatusCode:      http.StatusOK,
			Response:        `{"order":"123456","status":"PROCESSED","accrual":100.5}`,
			ExpectedAccrual: 100.5,
			ExpectedStatus:  models.OrderStatusProcessed,
			ExpectedError:   nil,
		},
		{
			TestName:        "Success. Order not found #2",
			OrderNumber:     "000000",
			StatusCode:      http.StatusNoContent,
			Response:        "",
			ExpectedAccrual: 0,
			ExpectedStatus:  models.OrderStatusInvalid,
			ExpectedError:   nil,
		},
		{
			TestName:        "Error. Too many requests #3",
			OrderNumber:     "654321",
			StatusCode:      http.StatusTooManyRequests,
			Response:        "",
			ExpectedAccrual: 0,
			ExpectedStatus:  models.OrderStatusProcessing,
			ExpectedError:   nil,
		},
		{
			TestName:        "Error. Accrual service error #4",
			OrderNumber:     "123123",
			StatusCode:      http.StatusInternalServerError,
			Response:        "",
			ExpectedAccrual: 0,
			ExpectedStatus:  "",
			ExpectedError:   ErrAccrualServiceUnavailable,
		},
		{
			TestName:        "Error. Invalid order status #5",
			OrderNumber:     "999999",
			StatusCode:      http.StatusOK,
			Response:        `{"order":"999999","status":"UNKNOWN","accrual":50.0}`,
			ExpectedAccrual: 0,
			ExpectedStatus:  "",
			ExpectedError:   errors.New("undefined status request UNKNOWN"),
		},
		{
			TestName:        "Error. Failed create request #6",
			OrderNumber:     "\x7f",
			ExpectedAccrual: 0,
			ExpectedStatus:  "",
			ExpectedError:   errors.New("failed to create new request"),
		},
		{
			TestName:        "Error. Failed decode response #7",
			OrderNumber:     "invalid",
			StatusCode:      http.StatusOK,
			Response:        `{"order":"123456","status":123}`,
			ExpectedAccrual: 0,
			ExpectedStatus:  "",
			ExpectedError:   errors.New("failed decode JSON response"),
		},
		{
			TestName:        "Error. Invalid URL request #8",
			OrderNumber:     "failure",
			StatusCode:      http.StatusNotFound,
			ExpectedAccrual: 0,
			ExpectedStatus:  "",
			ExpectedError:   errors.New("accrual service unavailable"),
		},
	}
	config := config.DefaultConfig()
	if err := logger.Initialize(config.Server.LogLevel); err != nil {
		logger.Panic(err)
	}
	defer logger.Sync()

	for _, tc := range testCases {
		t.Run(tc.TestName, func(t *testing.T) {
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.StatusCode == http.StatusTooManyRequests {
					w.Header().Set("Retry-After", "0")
				}
				w.WriteHeader(tc.StatusCode)
				if tc.Response != "" {
					_, err := w.Write([]byte(tc.Response))
					if err != nil {
						t.Fatalf("Failed to write test response: %v", err)
					}
				}
			}))
			defer testServer.Close()

			service := &Accrual{
				AccrualAddr: testServer.URL,
				Limiter:     rate.NewLimiter(rate.Inf, 1),
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			accrual, status, err := service.GetOrderAccrual(ctx, tc.OrderNumber)

			if accrual != tc.ExpectedAccrual {
				t.Errorf("Expected accrual: '%v', got: '%v'", tc.ExpectedAccrual, accrual)
			}
			if status != tc.ExpectedStatus {
				t.Errorf("Expected status: '%v', got: '%v'", tc.ExpectedStatus, status)
			}
			if tc.ExpectedError != nil {
				if err == nil {
					t.Errorf("Expected error: '%v', got: nil", tc.ExpectedError)
				} else if !strings.Contains(err.Error(), tc.ExpectedError.Error()) {
					t.Errorf("Expected error containing: '%v', got '%v'", tc.ExpectedError.Error(), err.Error())
				}
			} else if err != nil {
				t.Errorf("Expected no error, got: '%v'", err)
			}
		})
	}
}
