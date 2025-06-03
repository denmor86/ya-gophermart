package handlers

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/denmor86/ya-gophermart/internal/logger"
	"github.com/denmor86/ya-gophermart/internal/services"
)

// NewOrdersHandler — покупка пользователя
func NewOrdersHandler(s *services.Orders) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// получение данных о пользователе
		username, err := services.GetUsername(r.Context())
		if err != nil {
			logger.Warn("Failed to get username", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) == 0 {
			logger.Warn("Invalid body", err)
			http.Error(w, "Invalid body format", http.StatusBadRequest)
			return
		}
		defer func() {
			if err := r.Body.Close(); err != nil {
				logger.Error("Error to close body", err)
			}
		}()

		orderNumber := strings.TrimSpace(string(body))

		err = s.AddOrder(r.Context(), username, orderNumber)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrOrderAlreadyUploaded):
				w.WriteHeader(http.StatusOK)
				return
			case errors.Is(err, services.ErrOrderUploadedByAnother):
				http.Error(w, "Order number already uploaded by another user", http.StatusConflict)
				return
			default:
				logger.Error("Failed to add order", err)
				http.Error(w, "Server Error", http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusAccepted)
	})
}
