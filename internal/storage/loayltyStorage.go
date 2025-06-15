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
	InsertWithdrawal = `INSERT INTO LOYALTY (user_id, order_number, amount) 
							VALUES ($1, $2, $3) 
							ON CONFLICT (order_number) DO NOTHING
							RETURNING order_number;`
	GetWithdrawal = `SELECT order_number, user_id, amount, processed_at FROM LOYALTY WHERE user_id=$1 ORDER BY processed_at;`
)

type LoyaltyDatabase struct {
	DB *Database
}

// Создание хранилища
func NewLoyaltysStorage(db *Database) LoyaltysStorage {
	return &LoyaltyDatabase{DB: db}
}

// AddWithdrawal — добавление записи о выводе средств и обновление баланса пользователя в одной транзакции
func (s *LoyaltyDatabase) AddWithdrawal(ctx context.Context, loyalty models.WithdrawalData) error {
	// Начинаем транзакцию
	tx, err := s.DB.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Гарантированный откат при ошибке
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				logger.Error("Withdrawal. Rollback failed:", zap.Error(rbErr))
			}
		}
	}()

	// 1. Уменьшаем баланс пользователя (amount передаётся как отрицательное значение)
	_, err = tx.Exec(ctx, UpdateUserBalance, loyalty.Amount.Neg(), loyalty.UserID)
	if err != nil {
		logger.Error("Failed to update user balance", zap.Error(err))
		return fmt.Errorf("update balance: %w", err)
	}

	// 2. Добавляем запись о выводе
	var prevNumber string
	err = tx.QueryRow(
		ctx,
		InsertWithdrawal,
		loyalty.UserID,
		loyalty.OrderNumber,
		loyalty.Amount,
	).Scan(&prevNumber)

	// Обработка ошибок вставки
	if err == nil {
		// Успешная вставка → коммитим транзакцию
		if err = tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit failed: %w", err)
		}
		return nil
	}

	// Проверяем нарушение уникальности
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrAlreadyExists
	}

	// Все остальные ошибки
	return fmt.Errorf("insert withdrawal: %w", err)
}

func (s *LoyaltyDatabase) GetWithdrawals(ctx context.Context, userID string) ([]models.WithdrawalData, error) {
	var withdrawals []models.WithdrawalData
	rows, err := s.DB.Pool.Query(ctx, GetWithdrawal, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}
	for rows.Next() {
		var (
			orderNumber string
			userID      string
			amount      decimal.Decimal
			processedAt time.Time
		)
		err := rows.Scan(
			&orderNumber,
			&userID,
			&amount,
			&processedAt,
		)
		if err != nil {
			return withdrawals, fmt.Errorf("failed scan loyaltys data: %w", err)
		}
		withdrawals = append(withdrawals, models.WithdrawalData{
			OrderNumber: orderNumber,
			UserID:      userID,
			Amount:      amount,
			ProcessedAt: processedAt,
		})
	}
	return withdrawals, err
}
