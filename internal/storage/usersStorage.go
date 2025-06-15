package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/denmor86/ya-gophermart/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
)

const (
	InsertUser = `INSERT INTO USERS (id, login, password) 
						VALUES ($1, $2, $3) 
						ON CONFLICT (login) DO NOTHING
						RETURNING login;`
	GetUser = `SELECT id, password, login, balance FROM USERS WHERE login=$1;`

	GetUserBalance = `SELECT users.balance AS balance, COALESCE(SUM(LOYALTY.amount), 0) AS withdrawn
					  FROM 
					      USERS
					  LEFT JOIN 
					      LOYALTY ON USERS.id = LOYALTY.user_id
					  WHERE 
					      USERS.login = $1
					  GROUP BY 
					      USERS.balance;`
)

type UserDatabase struct {
	DB *Database
}

// Создание хранилища
func NewUsersStorage(db *Database) UsersStorage {
	return &UserDatabase{DB: db}
}

func (s *UserDatabase) GetUser(ctx context.Context, login string) (*models.UserData, error) {
	var (
		userID   string
		password string
		dbLogin  string
		balance  decimal.Decimal
	)
	err := s.DB.Pool.QueryRow(ctx, GetUser, login).Scan(&userID, &password, &dbLogin, &balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &models.UserData{
		UserID:       userID,
		Login:        dbLogin,
		PasswordHash: password,
		Balance:      balance,
	}, nil
}

func (s *UserDatabase) AddUser(ctx context.Context, login string, password string) error {
	var prevLogin string
	userID := uuid.New().String()

	err := s.DB.Pool.QueryRow(ctx, InsertUser, userID, login, password).Scan(&prevLogin)

	// Успешное добавление
	if err == nil {
		return nil
	}

	// Проверяем именно нарушение уникальности (код 23505)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrAlreadyExists
	}

	// Все остальные ошибки
	return fmt.Errorf("failed to add user: %w", err)
}

// GetUserBalance - Получение баланса и потраченных баллов пользователя
func (s *UserDatabase) GetUserBalance(ctx context.Context, login string) (*models.UserBalance, error) {
	var (
		current   float64
		withdrawn float64
	)

	err := s.DB.Pool.QueryRow(ctx, GetUserBalance, login).Scan(
		&current,
		&withdrawn,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return &models.UserBalance{
		Current:   current,
		Withdrawn: withdrawn,
	}, nil
}
