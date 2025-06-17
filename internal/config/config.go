package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env"
	"github.com/spf13/pflag"
)

type Arguments struct {
	ListenAddr             string        `env:"RUN_ADDRESS" envDefault:"localhost:8080"`
	LogLevel               string        `env:"LOG_LEVEL" envDefault:"info"`
	DatabaseDSN            string        `env:"DATABASE_URI" envDefault:""`
	JWTSecret              string        `env:"JWT_SECRET" envDefault:"secret"`
	AccrualAddr            string        `env:"ACCRUAL_SYSTEM_ADDRESS" envDefault:"http://localhost:8081"`
	BatchSize              int           `env:"WORKER_BATCH_SIZE" envDefault:"10"`
	PollInterval           time.Duration `env:"WORKER_POLL_INTERVAL" envDefault:"5s"`
	ProcessingTimeout      time.Duration `env:"WORKER_PROCESSING_TIMEOUT" envDefault:"10s"`
	CircuitBreakerTimeout  time.Duration `env:"WORKER_BREAKER_TIMEOUT" envDefault:"30s"`
	CircuitBreakerFailures uint32        `env:"WORKER_BREAKER_FAILURES" envDefault:"5"`
}

// ServerConfig модель настроек сервера
type ServerConfig struct {
	ListenAddr  string
	LogLevel    string
	JWTSecret   string
	DatabaseDSN string
}

// AccrualConfig модель настроек работы с сервисом  расчёта начислений баллов лояльности
type AccrualConfig struct {
	AccrualAddr            string
	BatchSize              int
	PollInterval           time.Duration
	ProcessingTimeout      time.Duration
	CircuitBreakerTimeout  time.Duration
	CircuitBreakerFailures uint32
}

// Config модель настроек сервиса
type Config struct {
	Server  ServerConfig
	Accrual AccrualConfig
}

func NewConfig() Config {

	var args Arguments
	if err := env.Parse(&args); err != nil {
		panic(fmt.Sprintf("Failed to parse enviroment var: %s", err.Error()))
	}

	var (
		server   = pflag.StringP("server", "a", args.ListenAddr, "Server listen address in a form host:port.")
		logLevel = pflag.StringP("log_level", "l", args.LogLevel, "Log level.")
		DSN      = pflag.StringP("dsn", "d", args.DatabaseDSN, "Database DSN")
		secret   = pflag.StringP("secret", "s", args.JWTSecret, "Secret to JWT")
		accrual  = pflag.StringP("accurual", "r", args.AccrualAddr, "Accurual listen address in a form host:port.")
	)
	pflag.Parse()

	return Config{
		Server: ServerConfig{
			ListenAddr:  *server,
			LogLevel:    *logLevel,
			DatabaseDSN: *DSN,
			JWTSecret:   *secret,
		},
		Accrual: AccrualConfig{
			AccrualAddr:            *accrual,
			BatchSize:              args.BatchSize,
			PollInterval:           args.PollInterval,
			ProcessingTimeout:      args.ProcessingTimeout,
			CircuitBreakerTimeout:  args.CircuitBreakerTimeout,
			CircuitBreakerFailures: args.CircuitBreakerFailures,
		},
	}
}

func DefaultConfig() Config {
	return Config{
		Server: ServerConfig{
			ListenAddr:  "localhost:8080",
			LogLevel:    "info",
			DatabaseDSN: "",
			JWTSecret:   "secret",
		},
		Accrual: AccrualConfig{
			AccrualAddr:            ":8081",
			BatchSize:              10,
			PollInterval:           5 * time.Second,
			ProcessingTimeout:      10 * time.Second,
			CircuitBreakerTimeout:  30 * time.Second,
			CircuitBreakerFailures: 5,
		},
	}
}
