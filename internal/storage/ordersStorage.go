package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/denmor86/ya-gophermart/internal/logger"
	"github.com/denmor86/ya-gophermart/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	GetOrder         = `SELECT user_id, status, created_at, accrual FROM ORDERS WHERE number=$1;`
	GetUserIDByOrder = `SELECT user_id FROM ORDERS WHERE number=$1;`
	InsertOrder      = `INSERT INTO ORDERS (number, user_id, status, accrual, retry_count, created_at, updated_at) 
						VALUES ($1, $2, $3, $4, $5, $6, $7) 
						ON CONFLICT (number) DO NOTHING
						RETURNING number;`
	GetOrders                = `SELECT number, status, created_at, accrual FROM ORDERS WHERE user_id=$1;`
	ClaimOrdersForProcessing = `UPDATE ORDERS 
								SET status = 'PROCESSING',
								    retry_count = retry_count + 1,
								    updated_at = NOW()
								WHERE number IN (
								    SELECT number FROM ORDERS 
								    WHERE status = 'NEW' OR status = 'REGISTERED' OR (status = 'PROCESSING' AND retry_count < 3)
								    ORDER BY created_at 
								    LIMIT $1
								    FOR UPDATE SKIP LOCKED
								)
								RETURNING number;`

	UpdateOrdersStatus = `UPDATE ORDERS 
						  SET 
						      status = $1,
						      accrual = $2,
						      retry_count = retry_count + 1,
						      updated_at = NOW()
						  WHERE number = $3;`
	UpdateUserBalance = `UPDATE USERS 
						  SET balance = balance + $1
						  WHERE id = $2;`
)

type OrderDatabase struct {
	DB *Database
}

// Создание хранилища
func NewOrdersStorage(db *Database) OrdersStorage {
	return &OrderDatabase{DB: db}
}

func (s *OrderDatabase) GetOrder(ctx context.Context, number string) (*models.OrderData, error) {
	var (
		userID     string
		status     string
		uploadedAt time.Time
		accrual    decimal.Decimal
	)

	err := s.DB.Pool.QueryRow(ctx, GetOrder, number).Scan(
		&userID,
		&status,
		&uploadedAt,
		&accrual,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return &models.OrderData{
		UserID:     userID,
		Status:     status,
		UploadedAt: uploadedAt,
		Accrual:    accrual,
	}, nil
}

func (s *OrderDatabase) GetOrders(ctx context.Context, userID string) ([]models.OrderData, error) {
	var orders []models.OrderData
	rows, err := s.DB.Pool.Query(ctx, GetOrders, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}
	for rows.Next() {
		var (
			number     string
			status     string
			uploadedAt time.Time
			accrual    decimal.Decimal
		)
		err := rows.Scan(
			&number,
			&status,
			&uploadedAt,
			&accrual,
		)
		if err != nil {
			return orders, fmt.Errorf("failed scan order data: %w", err)
		}
		orders = append(orders, models.OrderData{
			Number:     number,
			Status:     status,
			UploadedAt: uploadedAt,
			Accrual:    accrual})
	}
	return orders, err
}

func (s *OrderDatabase) ClaimOrdersForProcessing(ctx context.Context, count int) ([]string, error) {

	var numbers []string
	rows, err := s.DB.Pool.Query(ctx, ClaimOrdersForProcessing, count)
	if err != nil {
		return nil, fmt.Errorf("failed to get processing orders: %w", err)
	}
	for rows.Next() {

		var orderNumber string
		err := rows.Scan(&orderNumber)
		if err != nil {
			return numbers, fmt.Errorf("failed scan number for processing numbers: %w", err)
		}
		numbers = append(numbers, orderNumber)
	}
	return numbers, err
}

func (s *OrderDatabase) AddOrder(ctx context.Context, number string, userID string, createdAt time.Time) error {
	var prevNumber string
	err := s.DB.Pool.QueryRow(ctx, InsertOrder, number, userID, models.OrderStatusNew, 0, 0, createdAt, createdAt).Scan(&prevNumber)

	if err == nil {
		return nil
	}

	// Проверяем именно нарушение уникальности
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrAlreadyExists
	}

	// Все остальные ошибки
	return fmt.Errorf("failed to add order: %w", err)
}

// UpdateOrderAndBalance - Обновление статуса заказа и баланса пользователя в одной транзакции
func (s *OrderDatabase) UpdateOrderAndBalance(ctx context.Context, number string, status string, accrual decimal.Decimal) error {
	// Начинаем транзакцию
	tx, err := s.DB.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Гарантированный откат при ошибке
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				logger.Error("UpdateOrderAndBalance. rollback failed:", zap.Error(rbErr))
			}
		}
	}()

	// Обновляем статус заказа и начисление
	_, err = tx.Exec(ctx, UpdateOrdersStatus, status, accrual, number)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Обновляем баланс пользователя (только если есть начисление)
	if accrual.GreaterThan(decimal.Zero) {
		var userID string
		err = tx.QueryRow(ctx, GetUserIDByOrder, number).Scan(&userID)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}
		_, err = tx.Exec(ctx, UpdateUserBalance, accrual, userID)
		if err != nil {
			return fmt.Errorf("failed to update user balance: %w", err)
		}
	}

	// Если всё успешно - коммитим
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("UpdateOrderAndBalance. Commit failed: %w", err)
	}

	return nil
}
