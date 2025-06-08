package services

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/denmor86/ya-gophermart/internal/config"
	"github.com/denmor86/ya-gophermart/internal/logger"
	"github.com/denmor86/ya-gophermart/internal/models"
	"github.com/denmor86/ya-gophermart/internal/storage"
	"github.com/denmor86/ya-gophermart/internal/storage/mocks"
	"github.com/google/go-cmp/cmp"
	"go.uber.org/mock/gomock"
)

func TestOrderService_AddOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStorage := mocks.NewMockIStorage(ctrl)

	config := config.DefaultConfig()
	if err := logger.Initialize(config.LogLevel); err != nil {
		logger.Panic(err)
	}

	orders := NewOrders(mockStorage)

	testCases := []struct {
		TestName      string
		Login         string
		OrderNumber   string
		SetupMocks    func()
		ExpectedError error
	}{
		{
			TestName:    "Error. User not found #1",
			Login:       "mda",
			OrderNumber: "123456789",
			SetupMocks: func() {
				mockStorage.EXPECT().GetUser(gomock.Any(), "mda").Return(nil, fmt.Errorf("user %w", storage.ErrNotFound))
			},
			ExpectedError: fmt.Errorf("user %w", storage.ErrNotFound),
		},
		{
			TestName:    "Error. Failed get order #2",
			Login:       "mda",
			OrderNumber: "123456789",
			SetupMocks: func() {
				mockStorage.EXPECT().GetUser(gomock.Any(), "mda").Return(&models.UserData{UserUUID: "1"}, nil)
				mockStorage.EXPECT().GetOrder(gomock.Any(), "123456789").Return(nil, errors.New("failed to get order"))
			},
			ExpectedError: errors.New("failed to get order"),
		},
		{
			TestName:    "Error. Order already uploaded #3",
			Login:       "mda",
			OrderNumber: "123456789",
			SetupMocks: func() {
				mockStorage.EXPECT().GetUser(gomock.Any(), "mda").Return(&models.UserData{UserUUID: "1"}, nil)
				mockStorage.EXPECT().GetOrder(gomock.Any(), "123456789").Return(&models.OrderData{UserUUID: "1"}, nil)
			},
			ExpectedError: ErrOrderAlreadyUploaded,
		},
		{
			TestName:    "Error. Order already uploaded #4",
			Login:       "mda",
			OrderNumber: "123456789",
			SetupMocks: func() {
				mockStorage.EXPECT().GetUser(gomock.Any(), "mda").Return(&models.UserData{UserUUID: "1"}, nil)
				mockStorage.EXPECT().GetOrder(gomock.Any(), "123456789").Return(&models.OrderData{UserUUID: "2"}, nil)
			},
			ExpectedError: ErrOrderUploadedByAnother,
		},
		{
			TestName:    "Success. Order not found #5",
			Login:       "mda",
			OrderNumber: "123456789",
			SetupMocks: func() {
				mockStorage.EXPECT().GetUser(gomock.Any(), "mda").Return(&models.UserData{UserUUID: "1"}, nil)
				mockStorage.EXPECT().GetOrder(gomock.Any(), "123456789").Return(nil, fmt.Errorf("order %w", storage.ErrNotFound))
				mockStorage.EXPECT().AddOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			ExpectedError: nil,
		},
		{
			TestName:    "Error. Add order failure #6",
			Login:       "mda",
			OrderNumber: "123456789",
			SetupMocks: func() {
				mockStorage.EXPECT().GetUser(gomock.Any(), "mda").Return(&models.UserData{UserUUID: "1"}, nil)
				mockStorage.EXPECT().GetOrder(gomock.Any(), "123456789").Return(nil, fmt.Errorf("order %w", storage.ErrNotFound))
				mockStorage.EXPECT().AddOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("failed to add order"))
			},
			ExpectedError: errors.New("failed to add order"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.TestName, func(t *testing.T) {
			tc.SetupMocks()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := orders.AddOrder(ctx, tc.Login, tc.OrderNumber)

			if err != nil && tc.ExpectedError == nil {
				t.Errorf("Expected no error, got '%v'", err)
			} else if err == nil && tc.ExpectedError != nil {
				t.Errorf("Expected error, got none")
			} else if err != nil && err.Error() != tc.ExpectedError.Error() {
				t.Errorf("Expected error: '%v', got: '%v'", tc.ExpectedError, err)
			}
		})
	}
}

func TestOrderService_GetOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStorage := mocks.NewMockIStorage(ctrl)

	config := config.DefaultConfig()
	if err := logger.Initialize(config.LogLevel); err != nil {
		logger.Panic(err)
	}

	orders := NewOrders(mockStorage)

	testCases := []struct {
		Name           string
		Login          string
		SetupMocks     func()
		ExpectedError  error
		ExpectedOrders []models.OrderData
	}{
		{
			Name:  "Error. User not found #1",
			Login: "mda",
			SetupMocks: func() {
				mockStorage.EXPECT().GetUser(gomock.Any(), "mda").Return(nil, fmt.Errorf("user %w", storage.ErrNotFound))
			},
			ExpectedError:  fmt.Errorf("user %w", storage.ErrNotFound),
			ExpectedOrders: nil,
		},
		{
			Name:  "Error. Failed get orders #2",
			Login: "mda",
			SetupMocks: func() {
				mockStorage.EXPECT().GetUser(gomock.Any(), "mda").Return(&models.UserData{UserUUID: "1"}, nil)
				mockStorage.EXPECT().GetOrders(gomock.Any(), "1").Return(nil, errors.New("failed to get orders"))
			},
			ExpectedError:  errors.New("failed to get orders"),
			ExpectedOrders: nil,
		},
		{
			Name:  "Success. #3",
			Login: "mda",
			SetupMocks: func() {
				mockStorage.EXPECT().GetUser(gomock.Any(), "mda").Return(&models.UserData{UserUUID: "1"}, nil)
				mockStorage.EXPECT().GetOrders(gomock.Any(), "1").Return([]models.OrderData{
					{Number: "123456789", UserUUID: "1", Status: models.OrderStatusNew},
					{Number: "987654321", UserUUID: "1", Status: models.OrderStatusProcessed},
				}, nil)
			},
			ExpectedError: nil,
			ExpectedOrders: []models.OrderData{
				{Number: "123456789", UserUUID: "1", Status: models.OrderStatusNew},
				{Number: "987654321", UserUUID: "1", Status: models.OrderStatusProcessed},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tc.SetupMocks()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			orders, err := orders.GetOrders(ctx, tc.Login)

			if err != nil && tc.ExpectedError == nil {
				t.Errorf("Expected no error, got: '%v'", err)
			} else if err == nil && tc.ExpectedError != nil {
				t.Errorf("Expected error, got none")
			} else if err != nil && err.Error() != tc.ExpectedError.Error() {
				t.Errorf("Expected error^ '%v', got: '%v'", tc.ExpectedError, err)
			}
			diff := cmp.Diff(tc.ExpectedOrders, orders)
			if len(diff) != 0 {
				t.Errorf("expected orders mismatch:\n %s", diff)
			}
		})
	}
}

func TestOrderService_ClaimOrdersForProcessing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStorage := mocks.NewMockIStorage(ctrl)

	config := config.DefaultConfig()
	if err := logger.Initialize(config.LogLevel); err != nil {
		logger.Panic(err)
	}

	orders := NewOrders(mockStorage)

	testCases := []struct {
		Name                 string
		Size                 int
		SetupMocks           func()
		ExpectedError        error
		ExpectedOrderNumbers []string
	}{
		{
			Name: "Error. User not found #1",
			Size: -1,
			SetupMocks: func() {
				mockStorage.EXPECT().ClaimOrdersForProcessing(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("failed to get processing orders"))
			},
			ExpectedError:        fmt.Errorf("failed to get processing orders"),
			ExpectedOrderNumbers: nil,
		},
		{
			Name: "Success. #1",
			Size: 1,
			SetupMocks: func() {
				mockStorage.EXPECT().ClaimOrdersForProcessing(gomock.Any(), gomock.Any()).Return([]string{"123456789", "987654321"}, nil)
			},
			ExpectedError:        nil,
			ExpectedOrderNumbers: []string{"123456789", "987654321"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tc.SetupMocks()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			orders, err := orders.ClaimOrdersForProcessing(ctx, tc.Size)

			if err != nil && tc.ExpectedError == nil {
				t.Errorf("Expected no error, got: '%v'", err)
			} else if err == nil && tc.ExpectedError != nil {
				t.Errorf("Expected error, got none")
			} else if err != nil && err.Error() != tc.ExpectedError.Error() {
				t.Errorf("Expected error^ '%v', got: '%v'", tc.ExpectedError, err)
			}
			diff := cmp.Diff(tc.ExpectedOrderNumbers, orders)
			if len(diff) != 0 {
				t.Errorf("expected order numbers mismatch:\n %s", diff)
			}
		})
	}
}
