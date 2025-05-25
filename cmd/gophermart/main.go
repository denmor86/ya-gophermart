package main

import (
	"fmt"

	"github.com/denmor86/ya-gophermart/internal/app"
	"github.com/denmor86/ya-gophermart/internal/config"
	"github.com/denmor86/ya-gophermart/internal/logger"
)

func main() {
	// загрузка конфига
	config := config.NewConfig()
	// инициализация логгера
	if err := logger.Initialize(config.LogLevel); err != nil {
		panic(fmt.Sprintf("can't initialize logger: %s ", err.Error()))
	}
	defer logger.Sync()
	// создание маршутизатора
	app.Run(config)
}
