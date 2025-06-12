package storage

import (
	"context"
	"errors"
	"time"

	"github.com/denmor86/ya-gophermart/internal/models"
	"github.com/shopspring/decimal"
)

type IStorage interface {
	AddUser(ctx context.Context, login string, password string) error
	GetUser(ctx context.Context, login string) (*models.UserData, error)
	GetOrder(ctx context.Context, number string) (*models.OrderData, error)
	GetOrders(ctx context.Context, userID string) ([]models.OrderData, error)
	ClaimOrdersForProcessing(ctx context.Context, count int) ([]string, error)
	AddOrder(ctx context.Context, number string, userID string, createdAt time.Time) error
	GetUserBalance(ctx context.Context, login string) (*models.UserBalance, error)
	UpdateOrderAndBalance(ctx context.Context, number string, status string, accrual decimal.Decimal) error
	AddWithdrawal(ctx context.Context, loyalty models.WithdrawalData) error
	GetWithdrawals(ctx context.Context, userID string) ([]models.WithdrawalData, error)
	Ping(ctx context.Context) error
	Close() error
}

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrOrderNotFound = errors.New("order not found")

	ErrAlreadyExists = errors.New("already exists")
)
