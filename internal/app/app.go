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
)

func Run(config config.Config, storage storage.IStorage) {

	router := router.NewRouter(config, storage)

	server := &http.Server{
		Addr:    config.ListenAddr,
		Handler: router.HandleRouter(),
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info(
			"Starting server config:", config,
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("error listen server", err.Error())
		}
	}()

	<-stop
	logger.Info("Shutdown server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("error shutdown server", err.Error())
	}
	logger.Info("Server stopped")
}
