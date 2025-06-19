package main

import (
	"fmt"

	"github.com/denmor86/ya-gophermart/internal/app"
	"github.com/denmor86/ya-gophermart/internal/config"
	"github.com/denmor86/ya-gophermart/internal/logger"
	"github.com/denmor86/ya-gophermart/internal/storage"
	"github.com/pkg/errors"
)

func main() {
	// загрузка конфига
	config := config.NewConfig()
	// инициализация логгера
	if err := logger.Initialize(config.Server.LogLevel); err != nil {
		panic(fmt.Sprintf("can't initialize logger: %s ", err.Error()))
	}
	defer logger.Sync()

	database, err := storage.NewDatabase(config.Server.DatabaseDSN)
	if err != nil {
		panic(fmt.Sprintf("can't create database storage: %s ", errors.Cause(err).Error()))
	}
	if err = database.Initialize(); err != nil {
		panic(fmt.Sprintf("can't initialize database storage: %s ", errors.Cause(err).Error()))
	}
	defer database.Close()

	// создание маршутизатора
	app.Run(config, storage.NewStorage(database))
}
