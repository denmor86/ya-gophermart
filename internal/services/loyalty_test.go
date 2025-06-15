package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/denmor86/ya-gophermart/internal/config"
	"github.com/denmor86/ya-gophermart/internal/logger"
	"github.com/denmor86/ya-gophermart/internal/models"
	"github.com/denmor86/ya-gophermart/internal/storage"
	"github.com/denmor86/ya-gophermart/internal/storage/mocks"
	"github.com/google/go-cmp/cmp"
	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"
)

func TestLoyaltyService_GetBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLoyaltys := mocks.NewMockLoyaltysStorage(ctrl)
	mockUsers := mocks.NewMockUsersStorage(ctrl)

	config := config.DefaultConfig()
	if err := logger.Initialize(config.Server.LogLevel); err != nil {
		logger.Panic(err)
	}

	loyalty := NewLoyalty(mockLoyaltys, mockUsers)

	testCases := []struct {
		Name            string
		Login           string
		SetupMocks      func()
		ExpectedError   error
		ExpectedBalance *models.UserBalance
	}{
		{
			Name:  "Error. User not found #1",
			Login: "mda",
			SetupMocks: func() {
				mockUsers.EXPECT().GetUserBalance(gomock.Any(), "mda").Return(nil, storage.ErrUserNotFound)
			},
			ExpectedError:   storage.ErrUserNotFound,
			ExpectedBalance: nil,
		},
		{
			Name:  "Error. Failed get balance #2",
			Login: "mda",
			SetupMocks: func() {
				mockUsers.EXPECT().GetUserBalance(gomock.Any(), "mda").Return(nil, errors.New("failed to get orders"))
			},
			ExpectedError:   errors.New("failed to get orders"),
			ExpectedBalance: nil,
		},
		{
			Name:  "Success. #3",
			Login: "mda",
			SetupMocks: func() {
				mockUsers.EXPECT().GetUserBalance(gomock.Any(), "mda").Return(&models.UserBalance{Current: 10, Withdrawn: 5}, nil)
			},
			ExpectedError:   nil,
			ExpectedBalance: &models.UserBalance{Current: 10, Withdrawn: 5},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tc.SetupMocks()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			balance, err := loyalty.GetBalance(ctx, tc.Login)

			if err != nil && tc.ExpectedError == nil {
				t.Errorf("Expected no error, got: '%v'", err)
			} else if err == nil && tc.ExpectedError != nil {
				t.Errorf("Expected error, got none")
			} else if err != nil && err.Error() != tc.ExpectedError.Error() {
				t.Errorf("Expected error '%v', got: '%v'", tc.ExpectedError, err)
			}
			diff := cmp.Diff(tc.ExpectedBalance, balance)
			if len(diff) != 0 {
				t.Errorf("expected balance mismatch:\n %s", diff)
			}
		})
	}
}

func TestLoyaltyService_GetWithdrawals(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLoyaltys := mocks.NewMockLoyaltysStorage(ctrl)
	mockUsers := mocks.NewMockUsersStorage(ctrl)

	config := config.DefaultConfig()
	if err := logger.Initialize(config.Server.LogLevel); err != nil {
		logger.Panic(err)
	}

	loyalty := NewLoyalty(mockLoyaltys, mockUsers)

	testCases := []struct {
		Name                string
		Login               string
		SetupMocks          func()
		ExpectedError       error
		ExpectedWithdrawals []models.WithdrawalData
	}{
		{
			Name:  "Error. User not found #1",
			Login: "mda",
			SetupMocks: func() {
				mockUsers.EXPECT().GetUser(gomock.Any(), "mda").Return(nil, storage.ErrUserNotFound)
			},
			ExpectedError:       storage.ErrUserNotFound,
			ExpectedWithdrawals: nil,
		},
		{
			Name:  "Error. Failed get withdrawals #2",
			Login: "mda",
			SetupMocks: func() {
				mockUsers.EXPECT().GetUser(gomock.Any(), "mda").Return(&models.UserData{UserID: "1"}, nil)
				mockLoyaltys.EXPECT().GetWithdrawals(gomock.Any(), "1").Return(nil, errors.New("failed to get orders"))
			},
			ExpectedError:       errors.New("failed to get orders"),
			ExpectedWithdrawals: nil,
		},
		{
			Name:  "Success. #3",
			Login: "mda",
			SetupMocks: func() {
				mockUsers.EXPECT().GetUser(gomock.Any(), "mda").Return(&models.UserData{UserID: "1"}, nil)
				mockLoyaltys.EXPECT().GetWithdrawals(gomock.Any(), "1").Return([]models.WithdrawalData{
					{OrderNumber: "1", UserID: "1", Amount: decimal.NewFromInt(5)},
					{OrderNumber: "2", UserID: "1", Amount: decimal.NewFromInt(10)},
				}, nil)
			},
			ExpectedError: nil,
			ExpectedWithdrawals: []models.WithdrawalData{
				{OrderNumber: "1", UserID: "1", Amount: decimal.NewFromInt(5)},
				{OrderNumber: "2", UserID: "1", Amount: decimal.NewFromInt(10)},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tc.SetupMocks()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			withdrawal, err := loyalty.GetWithdrawals(ctx, tc.Login)

			if err != nil && tc.ExpectedError == nil {
				t.Errorf("Expected no error, got: '%v'", err)
			} else if err == nil && tc.ExpectedError != nil {
				t.Errorf("Expected error, got none")
			} else if err != nil && err.Error() != tc.ExpectedError.Error() {
				t.Errorf("Expected error '%v', got: '%v'", tc.ExpectedError, err)
			}
			diff := cmp.Diff(tc.ExpectedWithdrawals, withdrawal)
			if len(diff) != 0 {
				t.Errorf("expected withdrawal mismatch:\n %s", diff)
			}
		})
	}
}

func TestLoyaltyService_ProcessWithdraw(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLoyaltys := mocks.NewMockLoyaltysStorage(ctrl)
	mockUsers := mocks.NewMockUsersStorage(ctrl)

	config := config.DefaultConfig()
	if err := logger.Initialize(config.Server.LogLevel); err != nil {
		logger.Panic(err)
	}

	loyalty := NewLoyalty(mockLoyaltys, mockUsers)

	testCases := []struct {
		Name          string
		Login         string
		Number        string
		Sum           decimal.Decimal
		SetupMocks    func()
		ExpectedError error
	}{
		{
			Name:  "Error. User not found #1",
			Login: "mda",
			SetupMocks: func() {
				mockUsers.EXPECT().GetUser(gomock.Any(), "mda").Return(nil, storage.ErrUserNotFound)
			},
			ExpectedError: storage.ErrUserNotFound,
		},
		{
			Name:   "Error. Failed process withdrawals (invalid withdrawal amount) #2",
			Login:  "mda",
			Number: "1",
			Sum:    decimal.NewFromInt(-1),
			SetupMocks: func() {
				mockUsers.EXPECT().GetUser(gomock.Any(), "mda").Return(&models.UserData{UserID: "1", Balance: decimal.NewFromInt(10)}, nil)
			},
			ExpectedError: ErrInvalidWithdrawalAmount,
		},
		{
			Name:   "Error. Failed process withdrawals (insufficient funds for withdrawal) #3",
			Login:  "mda",
			Number: "1",
			Sum:    decimal.NewFromInt(11),
			SetupMocks: func() {
				mockUsers.EXPECT().GetUser(gomock.Any(), "mda").Return(&models.UserData{UserID: "1", Balance: decimal.NewFromInt(10)}, nil)
			},
			ExpectedError: ErrInsufficientFunds,
		},
		{
			Name:  "Error. Failed add withdrawals #4",
			Login: "mda",
			SetupMocks: func() {
				mockUsers.EXPECT().GetUser(gomock.Any(), "mda").Return(&models.UserData{UserID: "1", Balance: decimal.NewFromInt(10)}, nil)
				mockLoyaltys.EXPECT().AddWithdrawal(gomock.Any(), gomock.Any()).Return(errors.New("failed to get orders"))
			},
			ExpectedError: errors.New("failed to get orders"),
		},
		{
			Name:  "Success. #5",
			Login: "mda",
			SetupMocks: func() {
				mockUsers.EXPECT().GetUser(gomock.Any(), "mda").Return(&models.UserData{UserID: "1", Balance: decimal.NewFromInt(10)}, nil)
				mockLoyaltys.EXPECT().AddWithdrawal(gomock.Any(), gomock.Any()).Return(nil)
			},
			ExpectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tc.SetupMocks()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := loyalty.ProcessWithdraw(ctx, tc.Login, tc.Number, tc.Sum)

			if err != nil && tc.ExpectedError == nil {
				t.Errorf("Expected no error, got: '%v'", err)
			} else if err == nil && tc.ExpectedError != nil {
				t.Errorf("Expected error, got none")
			} else if err != nil && err.Error() != tc.ExpectedError.Error() {
				t.Errorf("Expected error '%v', got: '%v'", tc.ExpectedError, err)
			}
		})
	}
}
