package storage

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/denmor86/ya-gophermart/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type Database struct {
	Pool   *pgxpool.Pool
	Config *pgx.ConnConfig
	DSN    string
}

const (
	CheckExist     = `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname =$1)`
	CreateDatabase = `CREATE DATABASE %s`
	InsertUser     = `INSERT INTO USERS (uuid, login, password) 
						VALUES ($1, $2, $3) 
						ON CONFLICT (login) DO NOTHING
						RETURNING login;`
	GetUser     = `SELECT uuid, password, login FROM USERS WHERE login=$1;`
	GetOrder    = `SELECT user_uuid, status, created_at, accrual FROM ORDERS WHERE number=$1;`
	InsertOrder = `INSERT INTO ORDERS (number, user_uuid, status, accrual, retry_count, created_at, updated_at) 
						VALUES ($1, $2, $3, $4, $5, $6, $7) 
						ON CONFLICT (number) DO NOTHING
						RETURNING number;`
	GetOrders                = `SELECT user_uuid, status, created_at, accrual FROM ORDERS WHERE user_uuid=$1;`
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
							SET status = $1,
								accrual =$2
							    retry_count = retry_count + 1,
			                    updated_at = NOW()
							WHERE order_id = $3`
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
		userUUID string
		password string
		dbLogin  string
	)
	err := s.Pool.QueryRow(ctx, GetUser, login).Scan(&userUUID, &password, &dbLogin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user %w", ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &models.UserData{
		UserUUID:     userUUID,
		Login:        dbLogin,
		PasswordHash: password,
	}, nil
}

func (s *Database) AddUser(ctx context.Context, login string, password string) error {
	var prevLogin string
	userUUID := uuid.New().String()

	err := s.Pool.QueryRow(ctx, InsertUser, userUUID, login, password).Scan(&prevLogin)

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
		userUUID   string
		status     string
		uploadedAt time.Time
		accrual    float64
	)

	err := s.Pool.QueryRow(ctx, GetOrder, number).Scan(
		&userUUID,
		&status,
		&uploadedAt,
		&accrual,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("order %w", ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return &models.OrderData{
		UserUUID:   userUUID,
		Status:     status,
		UploadedAt: uploadedAt,
		Accrual:    accrual,
	}, nil
}

func (s *Database) GetOrders(ctx context.Context, userUUID string) ([]models.OrderData, error) {
	var orders []models.OrderData
	rows, err := s.Pool.Query(ctx, GetOrders, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}
	for rows.Next() {
		var (
			userUUID   string
			status     string
			uploadedAt time.Time
			accrual    float64
		)
		err := rows.Scan(
			&userUUID,
			&status,
			&uploadedAt,
			&accrual,
		)
		if err != nil {
			return orders, fmt.Errorf("failed scan order data: %w", err)
		}
		orders = append(orders, models.OrderData{
			UserUUID:   userUUID,
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

func (s *Database) AddOrder(ctx context.Context, number string, userUUID string, createdAt time.Time) error {
	var prevNumber string
	err := s.Pool.QueryRow(ctx, InsertOrder, number, userUUID, models.OrderStatusNew, 0, 0, createdAt, createdAt).Scan(&prevNumber)

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

func (s *Database) UpdateOrder(ctx context.Context, number string, status string, accrual float64) error {

	_, err := s.Pool.Exec(ctx, UpdateOrdersStatus, status, accrual, number)
	// Успешное добавление
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	return nil
}

func (s *Database) Ping(ctx context.Context) error {
	return s.Pool.Ping(ctx)
}
