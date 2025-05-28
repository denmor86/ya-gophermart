package storage

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type UniqueViolation struct {
	Message string
	Login   string
}

func (e *UniqueViolation) Error() string {
	return e.Message
}

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
	GetUserUUID     = `SELECT uuid FROM USERS WHERE login =$1;`
	GetUserPassword = `SELECT password FROM USERS WHERE login =$1;`
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

func (s *Database) GetUserUUID(ctx context.Context, login string) (string, error) {

	var userUUID string
	err := s.Pool.QueryRow(ctx, GetUserUUID, login).Scan(&userUUID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("user not found: %s", login)
		}
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	return userUUID, nil
}

func (s *Database) GetUserPassword(ctx context.Context, login string) (string, error) {

	var password string
	err := s.Pool.QueryRow(ctx, GetUserPassword, login).Scan(&password)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("user not found: %s", login)
		}
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	return password, nil
}
func (s *Database) AddUser(ctx context.Context, login string, password string) error {

	var prevLogin string
	userUUID := uuid.New().String()
	err := s.Pool.QueryRow(ctx, InsertUser, userUUID, login, password).Scan(&prevLogin)
	// добавили в базу, совпадений нет
	if err == nil {
		return nil
	}
	// ошибка добавления строки
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to add user: %w", err)
	}
	// есть совпадение пользователя
	return &UniqueViolation{Message: "User already exists", Login: prevLogin}
}

func (s *Database) Ping(ctx context.Context) error {
	return s.Pool.Ping(ctx)
}
