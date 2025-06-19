package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/denmor86/ya-gophermart/internal/client"
	mocks "github.com/denmor86/ya-gophermart/internal/client/mocks"
	"github.com/denmor86/ya-gophermart/internal/config"
	"github.com/denmor86/ya-gophermart/internal/logger"
	"github.com/denmor86/ya-gophermart/internal/models"
	"go.uber.org/mock/gomock"
)

func TestGetOrderAccrual(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockHTTPClient := mocks.NewMockHTTPClient(ctrl)

	config := config.DefaultConfig()
	if err := logger.Initialize(config.Server.LogLevel); err != nil {
		logger.Panic(err)
	}
	defer logger.Sync()

	testCases := []struct {
		TestName        string
		SetupMocks      func()
		OrderNumber     string
		ExpectedAccrual float64
		ExpectedStatus  string
		ExpectedError   error
	}{
		{
			TestName: "Success. Order processed #1",
			SetupMocks: func() {
				mockHTTPClient.EXPECT().Do(gomock.Any()).Return(&http.Response{
					Status:        "200 OK",
					StatusCode:    http.StatusOK,
					Body:          io.NopCloser(bytes.NewBufferString(`{"order":"123456","status":"PROCESSED","accrual":100.5}`)),
					ContentLength: int64(len(`{"order":"123456","status":"PROCESSED","accrual":100.5}`)),
					Header:        make(http.Header),
				}, nil)
			},
			OrderNumber:     "123456",
			ExpectedAccrual: 100.5,
			ExpectedStatus:  models.OrderStatusProcessed,
			ExpectedError:   nil,
		},
		{
			TestName: "Success. Order not found #2",
			SetupMocks: func() {
				mockHTTPClient.EXPECT().Do(gomock.Any()).Return(&http.Response{
					Status:        "204",
					StatusCode:    http.StatusNoContent,
					Body:          io.NopCloser(bytes.NewBufferString("")),
					ContentLength: int64(len("")),
					Header:        make(http.Header),
				}, nil)
			},
			OrderNumber:     "000000",
			ExpectedAccrual: 0,
			ExpectedStatus:  models.OrderStatusInvalid,
			ExpectedError:   client.ErrOrderNotRegistered,
		},
		{
			TestName: "Error. Too many requests #3",
			SetupMocks: func() {
				mockHTTPClient.EXPECT().Do(gomock.Any()).Return(&http.Response{
					Status:     "429 Too Many Requests",
					StatusCode: http.StatusTooManyRequests,
					Body:       io.NopCloser(bytes.NewBufferString("No more than N requests per minute allowed")),
					Header: http.Header{
						"Retry-After":  []string{"120"},
						"Content-Type": []string{"application/json"},
					},
				}, nil)
			},
			OrderNumber:     "654321",
			ExpectedAccrual: 0,
			ExpectedStatus:  models.OrderStatusProcessing,
			ExpectedError:   nil,
		},
		{
			TestName: "Error. Accrual service error #4",
			SetupMocks: func() {
				mockHTTPClient.EXPECT().Do(gomock.Any()).Return(&http.Response{
					Status:     "500",
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBufferString("")),
					Header:     make(http.Header),
				}, nil)
			},
			OrderNumber:     "123123",
			ExpectedAccrual: 0,
			ExpectedStatus:  models.OrderStatusInvalid,
			ExpectedError:   client.ErrServiceUnavailable,
		},
		{
			TestName: "Error. Invalid order status #5",
			SetupMocks: func() {
				mockHTTPClient.EXPECT().Do(gomock.Any()).Return(&http.Response{
					Status:     "200 OK",
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"order":"999999","status":"UNKNOWN","accrual":50.0}`)),
					Header:     make(http.Header),
				}, nil)
			},
			OrderNumber:     "999999",
			ExpectedAccrual: 0,
			ExpectedStatus:  "",
			ExpectedError:   errors.New("undefined status request UNKNOWN"),
		},
		{
			TestName: "Error. Failed decode response #6",
			SetupMocks: func() {
				mockHTTPClient.EXPECT().Do(gomock.Any()).Return(&http.Response{
					Status:        "200 OK",
					StatusCode:    http.StatusOK,
					Body:          io.NopCloser(bytes.NewBufferString(`{"orders":"999999","statuses":"UNKNOWN","accrualas":50.0}`)),
					ContentLength: int64(len(`{"orders":"999999","statuses":"UNKNOWN","accrualas":50.0}`)),
					Header:        make(http.Header),
				}, nil)
			},
			OrderNumber:     "invalid",
			ExpectedAccrual: 0,
			ExpectedStatus:  "",
			ExpectedError:   fmt.Errorf("undefined status request %s", ""),
		},
		{
			TestName: "Error. Invalid URL request #7",
			SetupMocks: func() {
				mockHTTPClient.EXPECT().Do(gomock.Any()).Return(&http.Response{
					Status:        "404",
					StatusCode:    http.StatusNotFound,
					Body:          io.NopCloser(bytes.NewBufferString("123")),
					ContentLength: int64(len("123")),
					Header:        make(http.Header),
				}, nil)
			},
			OrderNumber:     "failure",
			ExpectedAccrual: 0,
			ExpectedStatus:  models.OrderStatusInvalid,
			ExpectedError:   client.ErrServiceUnavailable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.TestName, func(t *testing.T) {
			tc.SetupMocks()

			service := &AccrualService{
				Client:  client.NewClient("", mockHTTPClient),
				Limiter: client.NewRateLimiter(),
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
