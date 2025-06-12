package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/denmor86/ya-gophermart/internal/helpers"
	"github.com/denmor86/ya-gophermart/internal/logger"
	"github.com/denmor86/ya-gophermart/internal/models"
	"github.com/denmor86/ya-gophermart/internal/services"
	"go.uber.org/zap"
)

// OrdersHandler — обработчик совершения покупки пользователем
func OrdersHandler(s services.OrdersService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// получение данных о пользователе
		username, err := services.GetUsername(r.Context())
		if err != nil {
			logger.Warn("Failed to get username:", zap.Error(err))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) == 0 {
			logger.Warn("Invalid body:", zap.Error(err))
			http.Error(w, "Invalid body format", http.StatusBadRequest)
			return
		}
		defer func() {
			if err := r.Body.Close(); err != nil {
				logger.Error("Error to close body:", zap.Error(err))
			}
		}()

		orderNumber := strings.TrimSpace(string(body))

		if !helpers.CheckNumber(orderNumber) {
			logger.Warn("Invalid order number format", orderNumber)
			http.Error(w, "Invalid order number format", http.StatusUnprocessableEntity)
			return
		}

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
				logger.Error("Failed to add order:", zap.Error(err))
				http.Error(w, "Server Error", http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusAccepted)
	})
}

// GetOrdersHandler — получение списка покупок пользователя
func GetOrdersHandler(s services.OrdersService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// получение данных о пользователе
		username, err := services.GetUsername(r.Context())
		if err != nil {
			logger.Warn("Failed to get username:", zap.Error(err))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		orders, err := s.GetOrders(r.Context(), username)
		if err != nil {
			logger.Error("Failed to get order:", zap.Error(err))
			http.Error(w, "Server Error", http.StatusInternalServerError)
			return
		}
		if len(orders) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		var response []models.OrderResponse
		for _, order := range orders {
			item := models.OrderResponse{
				Number:     order.Number,
				Status:     order.Status,
				UploadedAt: order.UploadedAt.Format(time.RFC3339),
			}
			if order.Status == models.OrderStatusProcessed {
				item.Accrual = order.Accrual
			}
			response = append(response, item)
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			logger.Error("Failed to encode JSON response:", zap.Error(err))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})
}
