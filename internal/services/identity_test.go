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
	"golang.org/x/crypto/bcrypt"

	"go.uber.org/mock/gomock"
)

func TestNewIdentityService(t *testing.T) {
	t.Run("Identity_CreatesService", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockUserRepo := mocks.NewMockIStorage(ctrl)

		config := config.DefaultConfig()
		identity := NewIdentity(config, mockUserRepo)
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
	if err := logger.Initialize(config.LogLevel); err != nil {
		logger.Panic(err)
	}

	testCases := []struct {
		name          string
		setupMocks    func()
		expectedError error
		user          models.UserRequest
	}{
		{
			name: "Register User: Success #1",
			setupMocks: func() {
				mockStorage.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(nil, nil)
				mockStorage.EXPECT().AddUser(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
			user:          models.UserRequest{Login: "mda", Password: "test_pass"},
		},
		{
			name: "Register User: ErrUserAlreadyExists #2",
			setupMocks: func() {
				mockStorage.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(&models.UserData{Login: "mda"}, nil)
			},
			expectedError: ErrUserAlreadyExists,
			user:          models.UserRequest{Login: "mda", Password: "test_pass"},
		},
		{
			name: "Register User: Undefined error #2",
			setupMocks: func() {
				mockStorage.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(nil, nil)
				mockStorage.EXPECT().AddUser(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("failed to add user"))
			},
			expectedError: errors.New("failed to add user"),
			user:          models.UserRequest{Login: "mda", Password: "test_pass"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMocks()

			identity := NewIdentity(config, mockStorage)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := identity.RegisterUser(ctx, tc.user)

			if err != nil && tc.expectedError == nil {
				t.Errorf("Expected no error, got: '%v'", err)
			} else if err == nil && tc.expectedError != nil {
				t.Errorf("Expected error, got none")
			} else if err != nil && err.Error() != tc.expectedError.Error() {
				t.Errorf("Expected error: '%v', got: '%v'", tc.expectedError, err)
			}
		})
	}
}

func TestAuthenticateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStorage := mocks.NewMockIStorage(ctrl)

	config := config.DefaultConfig()
	if err := logger.Initialize(config.LogLevel); err != nil {
		logger.Panic(err)
	}
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("test_pass"), bcrypt.DefaultCost)

	testCases := []struct {
		name          string
		mockReturn    func(ctx context.Context, login string) (*models.UserData, error)
		user          models.UserRequest
		expectedAuth  bool
		expectedError error
	}{
		{
			name: "AuthenticateUser Success #1",
			mockReturn: func(ctx context.Context, login string) (*models.UserData, error) {
				return &models.UserData{UserUUID: "1", Login: "mda", PasswordHash: string(passwordHash)}, nil
			},
			user:          models.UserRequest{Login: "mda", Password: "test_pass"},
			expectedAuth:  true,
			expectedError: nil,
		},
		{
			name: "AuthenticateUser UserNotFound #2",
			mockReturn: func(ctx context.Context, login string) (*models.UserData, error) {
				return nil, fmt.Errorf("user %w", storage.ErrNotFound)
			},
			user:          models.UserRequest{Login: "mda", Password: "test_pass"},
			expectedAuth:  false,
			expectedError: errors.New("user not found"),
		},
		{
			name: "AuthenticateUser InvalidPassword #3",
			mockReturn: func(ctx context.Context, login string) (*models.UserData, error) {
				return &models.UserData{UserUUID: "1", Login: "mda", PasswordHash: string("test_pass")}, nil
			},
			user:          models.UserRequest{Login: "mda", Password: "test_pass"},
			expectedAuth:  false,
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockStorage.EXPECT().GetUser(gomock.Any(), gomock.Any()).DoAndReturn(tc.mockReturn)

			identity := NewIdentity(config, mockStorage)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			authenticated, err := identity.AuthenticateUser(ctx, tc.user)

			if authenticated != tc.expectedAuth {
				t.Errorf("Expected authenticated %v, got %v", tc.expectedAuth, authenticated)
			}

			if err != nil && tc.expectedError == nil {
				t.Errorf("Expected no error, got: '%v'", err)
			} else if err == nil && tc.expectedError != nil {
				t.Errorf("Expected error, got none")
			} else if err != nil && err.Error() != tc.expectedError.Error() {
				t.Errorf("Expected error: '%v', got: '%v'", tc.expectedError, err)
			}
		})
	}
}
