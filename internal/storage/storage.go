package storage

import (
	"context"
	"errors"
	"time"

	"github.com/denmor86/ya-gophermart/internal/models"
)

type IStorage interface {
	AddUser(ctx context.Context, login string, password string) error
	GetUser(ctx context.Context, login string) (*models.UserData, error)
	GetOrder(ctx context.Context, number string) (*models.OrderData, error)
	GetOrders(ctx context.Context, userUUID string) ([]models.OrderData, error)
	ClaimOrdersForProcessing(ctx context.Context, count int) ([]string, error)
	AddOrder(ctx context.Context, number string, userUUID string, uploadetAt time.Time) error
	Ping(ctx context.Context) error
	Close() error
}

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)
