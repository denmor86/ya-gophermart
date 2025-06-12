package storage

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/denmor86/ya-gophermart/internal/logger"
	"github.com/denmor86/ya-gophermart/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type Database struct {
	Pool   *pgxpool.Pool
	Config *pgx.ConnConfig
	DSN    string
}

const (
	CheckExist     = `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname =$1)`
	CreateDatabase = `CREATE DATABASE %s`
	InsertUser     = `INSERT INTO USERS (id, login, password) 
						VALUES ($1, $2, $3) 
						ON CONFLICT (login) DO NOTHING
						RETURNING login;`
	GetUser          = `SELECT id, password, login, balance FROM USERS WHERE login=$1;`
	GetOrder         = `SELECT user_id, status, created_at, accrual FROM ORDERS WHERE number=$1;`
	GetUserIdByOrder = `SELECT user_id FROM ORDERS WHERE number=$1;`
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
	GetUserBalance = `SELECT users.balance AS balance, COALESCE(SUM(LOYALTY.amount), 0) AS withdrawn
					  FROM 
					      USERS
					  LEFT JOIN 
					      LOYALTY ON USERS.id = LOYALTY.user_id
					  WHERE 
					      USERS.login = $1
					  GROUP BY 
					      USERS.balance;`
	UpdateUserBalance = `UPDATE USERS 
						  SET balance = $1,
						  WHERE id = $2;`
	InsertWithdrawal = `INSERT INTO LOYALTY (user_id, order_number, amount) 
							VALUES ($1, $2, $3) 
							ON CONFLICT (order_number) DO NOTHING
							RETURNING order_number;`
	GetWithdrawal = `SELECT order_number, user_id, amount, processed_at FROM ORDERS WHERE user_id=$1 ORDER BY created_at;`
)

// Создание хранилища
func NewDatabaseStorage(dsn string) (*Database, error) {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}
	return &Database{Pool: pool, Config: cfg.ConnConfig, DSN: dsn}, nil
}

// Инициализация хранилища (создание БД, миграция)
func (s *Database) Initialize() error {

	if err := s.CreateDatabase(context.Background()); err != nil {
		return fmt.Errorf("error create database: %w", err)
	}
	if err := Migration(s.DSN); err != nil {
		return fmt.Errorf("error migrate database: %w", err)
	}

	return nil
}

//go:embed migrations/*.sql
var embedMigrations embed.FS

func Migration(DatabaseDSN string) error {

	db, err := sql.Open("pgx", DatabaseDSN)
	if err != nil {
		return fmt.Errorf("open db error: %w ", err)
	}
	defer db.Close()
	// используется для внутренней файловой системы (загруженные ресурсы)
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose set dialect error: %w ", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("goose run migrations error:  %w ", err)
	}
	return nil
}

func (s *Database) Close() error {
	s.Pool.Close()
	return nil
}

func (s *Database) CreateDatabase(ctx context.Context) error {
	// goose не умеет создавать БД
	conn, err := pgx.ConnectConfig(ctx, s.Config)
	if err != nil {
		// если не получилось соединиться с БД из строки подключения
		// пробуем использовать дефолтную БД
		cfg := s.Config.Copy()
		cfg.Database = `postgres`
		conn, err = pgx.ConnectConfig(ctx, cfg)
		if err != nil {
			return fmt.Errorf("failed to connect database: %w", err)
		}
		var exist bool
		err = conn.QueryRow(ctx, CheckExist, s.Config.Database).Scan(&exist)
		if err != nil {
			return fmt.Errorf("failed to check database exists: %w", err)
		}
		if !exist {
			_, err = conn.Exec(ctx, fmt.Sprintf(CreateDatabase, s.Config.Database))
			if err != nil {
				return fmt.Errorf("failed to create database: %w", err)
			}
		}
	}
	defer conn.Close(ctx)
	return nil
}

func (s *Database) GetUser(ctx context.Context, login string) (*models.UserData, error) {
	var (
		userID   string
		password string
		dbLogin  string
		balance  decimal.Decimal
	)
	err := s.Pool.QueryRow(ctx, GetUser, login).Scan(&userID, &password, &dbLogin, &balance)
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

func (s *Database) AddUser(ctx context.Context, login string, password string) error {
	var prevLogin string
	userID := uuid.New().String()

	err := s.Pool.QueryRow(ctx, InsertUser, userID, login, password).Scan(&prevLogin)

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

func (s *Database) GetOrder(ctx context.Context, number string) (*models.OrderData, error) {
	var (
		userID     string
		status     string
		uploadedAt time.Time
		accrual    decimal.Decimal
	)

	err := s.Pool.QueryRow(ctx, GetOrder, number).Scan(
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

func (s *Database) GetOrders(ctx context.Context, userID string) ([]models.OrderData, error) {
	var orders []models.OrderData
	rows, err := s.Pool.Query(ctx, GetOrders, userID)
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

func (s *Database) ClaimOrdersForProcessing(ctx context.Context, count int) ([]string, error) {

	var numbers []string
	rows, err := s.Pool.Query(ctx, ClaimOrdersForProcessing, count)
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

func (s *Database) AddOrder(ctx context.Context, number string, userID string, createdAt time.Time) error {
	var prevNumber string
	err := s.Pool.QueryRow(ctx, InsertOrder, number, userID, models.OrderStatusNew, 0, 0, createdAt, createdAt).Scan(&prevNumber)

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
func (s *Database) UpdateOrderAndBalance(ctx context.Context, number string, status string, accrual decimal.Decimal) error {
	// Начинаем транзакцию
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
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
		err = tx.QueryRow(ctx, GetUserIdByOrder, number).Scan(&userID)
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

// GetUserBalance - Получение баланса и потраченных баллов пользователя
func (s *Database) GetUserBalance(ctx context.Context, login string) (*models.UserBalance, error) {
	var (
		current   decimal.Decimal
		withdrawn decimal.Decimal
	)

	err := s.Pool.QueryRow(ctx, GetUserBalance, login).Scan(
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

// AddWithdrawal — добавление записи о выводе средств и обновление баланса пользователя в одной транзакции
func (s *Database) AddWithdrawal(ctx context.Context, loyalty models.WithdrawalData) error {
	// Начинаем транзакцию
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
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
		loyalty.OrderNumber,
		loyalty.UserID,
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

func (s *Database) GetWithdrawals(ctx context.Context, userID string) ([]models.WithdrawalData, error) {
	var withdrawals []models.WithdrawalData
	rows, err := s.Pool.Query(ctx, GetWithdrawal, userID)
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

func (s *Database) Ping(ctx context.Context) error {
	return s.Pool.Ping(ctx)
}
