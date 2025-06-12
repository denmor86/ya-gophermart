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
	"golang.org/x/crypto/bcrypt"

	"go.uber.org/mock/gomock"
)

func TestNewIdentityService(t *testing.T) {
	t.Run("Identity. CreatesService", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockUserRepo := mocks.NewMockIStorage(ctrl)

		config := config.DefaultConfig()
		identity := NewIdentity(config.Server.JWTSecret, mockUserRepo)
		baseService, ok := identity.(*Identity)
		if !ok {
			t.Fatalf("Expected *Identity, got: '%T'", identity)
		}
		if baseService == nil || baseService.JWTAuth == nil {
			t.Errorf("Expected Identity to be initialized with JWTAuth")
		}
		if baseService.Storage != mockUserRepo {
			t.Errorf("Expected Identity to be initialized with provided storage")
		}
	})
}

func TestRegisterUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStorage := mocks.NewMockIStorage(ctrl)

	config := config.DefaultConfig()
	if err := logger.Initialize(config.Server.LogLevel); err != nil {
		logger.Panic(err)
	}

	testCases := []struct {
		TestName      string
		SetupMocks    func()
		ExpectedError error
		User          models.UserRequest
	}{
		{
			TestName: "Success. Register user #1",
			SetupMocks: func() {
				mockStorage.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(nil, nil)
				mockStorage.EXPECT().AddUser(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			ExpectedError: nil,
			User:          models.UserRequest{Login: "mda", Password: "test_pass"},
		},
		{
			TestName: "Error. Register user already exists #2",
			SetupMocks: func() {
				mockStorage.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(&models.UserData{Login: "mda"}, nil)
			},
			ExpectedError: ErrUserAlreadyExists,
			User:          models.UserRequest{Login: "mda", Password: "test_pass"},
		},
		{
			TestName: "Error. Register user undefined error #3",
			SetupMocks: func() {
				mockStorage.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(nil, nil)
				mockStorage.EXPECT().AddUser(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("failed to add user"))
			},
			ExpectedError: errors.New("failed to add user"),
			User:          models.UserRequest{Login: "mda", Password: "test_pass"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.TestName, func(t *testing.T) {
			tc.SetupMocks()

			identity := NewIdentity(config.Server.JWTSecret, mockStorage)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := identity.RegisterUser(ctx, tc.User)

			if err != nil && tc.ExpectedError == nil {
				t.Errorf("Expected no error, got: '%v'", err)
			} else if err == nil && tc.ExpectedError != nil {
				t.Errorf("Expected error, got none")
			} else if err != nil && err.Error() != tc.ExpectedError.Error() {
				t.Errorf("Expected error: '%v', got: '%v'", tc.ExpectedError, err)
			}
		})
	}
}

func TestAuthenticateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStorage := mocks.NewMockIStorage(ctrl)

	config := config.DefaultConfig()
	if err := logger.Initialize(config.Server.LogLevel); err != nil {
		logger.Panic(err)
	}
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("test_pass"), bcrypt.DefaultCost)

	testCases := []struct {
		TestName      string
		mockReturn    func(ctx context.Context, login string) (*models.UserData, error)
		User          models.UserRequest
		expectedAuth  bool
		ExpectedError error
	}{
		{
			TestName: "AuthenticateUser Success #1",
			mockReturn: func(ctx context.Context, login string) (*models.UserData, error) {
				return &models.UserData{UserID: "1", Login: "mda", PasswordHash: string(passwordHash)}, nil
			},
			User:          models.UserRequest{Login: "mda", Password: "test_pass"},
			expectedAuth:  true,
			ExpectedError: nil,
		},
		{
			TestName: "AuthenticateUser UserNotFound #2",
			mockReturn: func(ctx context.Context, login string) (*models.UserData, error) {
				return nil, storage.ErrUserNotFound
			},
			User:          models.UserRequest{Login: "mda", Password: "test_pass"},
			expectedAuth:  false,
			ExpectedError: storage.ErrUserNotFound,
		},
		{
			TestName: "AuthenticateUser InvalidPassword #3",
			mockReturn: func(ctx context.Context, login string) (*models.UserData, error) {
				return &models.UserData{UserID: "1", Login: "mda", PasswordHash: string("test_pass")}, nil
			},
			User:          models.UserRequest{Login: "mda", Password: "test_pass"},
			expectedAuth:  false,
			ExpectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.TestName, func(t *testing.T) {
			mockStorage.EXPECT().GetUser(gomock.Any(), gomock.Any()).DoAndReturn(tc.mockReturn)

			identity := NewIdentity(config.Server.JWTSecret, mockStorage)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			authenticated, err := identity.AuthenticateUser(ctx, tc.User)

			if authenticated != tc.expectedAuth {
				t.Errorf("Expected authenticated %v, got %v", tc.expectedAuth, authenticated)
			}

			if err != nil && tc.ExpectedError == nil {
				t.Errorf("Expected no error, got: '%v'", err)
			} else if err == nil && tc.ExpectedError != nil {
				t.Errorf("Expected error, got none")
			} else if err != nil && err.Error() != tc.ExpectedError.Error() {
				t.Errorf("Expected error: '%v', got: '%v'", tc.ExpectedError, err)
			}
		})
	}
}
