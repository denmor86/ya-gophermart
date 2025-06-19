package storage

import (
	"context"
	"errors"
	"time"

	"github.com/denmor86/ya-gophermart/internal/models"
	"github.com/shopspring/decimal"
)

type UsersStorage interface {
	AddUser(ctx context.Context, login string, password string) error
	GetUser(ctx context.Context, login string) (*models.UserData, error)
	GetUserBalance(ctx context.Context, login string) (*models.UserBalance, error)
}

type OrdersStorage interface {
	GetOrder(ctx context.Context, number string) (*models.OrderData, error)
	GetOrders(ctx context.Context, userID string) ([]models.OrderData, error)
	ClaimOrdersForProcessing(ctx context.Context, count int) ([]string, error)
	AddOrder(ctx context.Context, number string, userID string, createdAt time.Time) error
	UpdateOrderAndBalance(ctx context.Context, number string, status string, accrual decimal.Decimal) error
}

type LoyaltysStorage interface {
	AddWithdrawal(ctx context.Context, loyalty models.WithdrawalData) error
	GetWithdrawals(ctx context.Context, userID string) ([]models.WithdrawalData, error)
}

type Storage struct {
	Users    UsersStorage
	Orders   OrdersStorage
	Loyaltys LoyaltysStorage
}

// Создание хранилища
func NewStorage(db *Database) Storage {
	return Storage{Users: NewUsersStorage(db), Orders: NewOrdersStorage(db), Loyaltys: NewLoyaltysStorage(db)}
}

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrOrderNotFound = errors.New("order not found")

	ErrAlreadyExists = errors.New("already exists")
)
