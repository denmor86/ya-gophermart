package config

import (
	"fmt"

	"github.com/caarlos0/env"
	"github.com/spf13/pflag"
)

type Config struct {
	ListenAddr  string `env:"SERVER_ADDRESS" envDefault:"localhost:8080"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
	DatabaseDSN string `env:"DATABASE_DSN" envDefault:""`
	JWTSecret   string `env:"JWT_SECRET" envDefault:"secret"`
	AccrualAddr string `env:"ACCRUAL_SYSTEM_ADDRESS" envDefault:"localhost:8081"`
}

func NewConfig() Config {

	var config Config
	if err := env.Parse(&config); err != nil {
		panic(fmt.Sprintf("Failed to parse enviroment var: %s", err.Error()))
	}

	var (
		server   = pflag.StringP("server", "a", config.ListenAddr, "Server listen address in a form host:port.")
		logLevel = pflag.StringP("log_level", "l", config.LogLevel, "Log level.")
		DSN      = pflag.StringP("dsn", "d", config.DatabaseDSN, "Database DSN")
		secret   = pflag.StringP("secret", "s", config.JWTSecret, "Secret to JWT")
		accrual  = pflag.StringP("accurual", "r", config.AccrualAddr, "Accurual listen address in a form host:port.")
	)
	pflag.Parse()

	return Config{
		ListenAddr:  *server,
		LogLevel:    *logLevel,
		DatabaseDSN: *DSN,
		JWTSecret:   *secret,
		AccrualAddr: *accrual,
	}
}

func DefaultConfig() Config {
	return Config{
		ListenAddr:  "localhost:8080",
		LogLevel:    "info",
		DatabaseDSN: "",
		JWTSecret:   "secret",
		AccrualAddr: "localhost:8081",
	}
}
