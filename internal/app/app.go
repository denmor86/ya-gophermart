package app

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/denmor86/ya-gophermart/internal/config"
	"github.com/denmor86/ya-gophermart/internal/logger"
	"github.com/denmor86/ya-gophermart/internal/network/router"
	"github.com/denmor86/ya-gophermart/internal/storage"
	"github.com/denmor86/ya-gophermart/internal/worker"
	"go.uber.org/zap"
)

func Run(config config.Config, storage storage.Storage) {

	router := router.NewRouter(config, storage)

	server := &http.Server{
		Addr:    config.Server.ListenAddr,
		Handler: router.HandleRouter(),
	}
	// Создание и запуск воркера
	worker := worker.NewOrderWorker(router.Orders, config.Accrual)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker.Start(ctx)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info(
			"Starting server config:", config,
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("error listen server:", zap.Error(err))
		}
	}()

	<-stop
	logger.Info("Shutdown server")
	worker.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("error shutdown server:", zap.Error(err))
	}
	logger.Info("Server stopped")
}
