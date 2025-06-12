package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/denmor86/ya-gophermart/internal/helpers"
	"github.com/denmor86/ya-gophermart/internal/logger"
	"github.com/denmor86/ya-gophermart/internal/models"
	"github.com/denmor86/ya-gophermart/internal/services"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// GetOrdersHandler — получение списка покупок пользователя
func GetUserBalanceHandler(l services.LoyaltyService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// получение данных о пользователе
		username, err := helpers.GetUsername(r.Context())
		if err != nil {
			logger.Warn("Failed to get username:", zap.Error(err))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		balance, err := l.GetBalance(r.Context(), username)
		if err != nil {
			logger.Error("Failed to get user balance:", zap.Error(err))
			http.Error(w, "Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(balance)
		if err != nil {
			logger.Error("Failed to encode JSON response:", zap.Error(err))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})
}

// WithdrawHandler — Запрос на списание средств
func WithdrawHandler(l services.LoyaltyService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// получение данных о пользователе
		username, err := helpers.GetUsername(r.Context())
		if err != nil {
			logger.Warn("Failed to get username:", zap.Error(err))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		var req models.WithdrawalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Warn("Invalid request format:", zap.Error(err))
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}
		if !helpers.CheckNumber(req.OrderNumber) {
			logger.Warn("Invalid order number format", req.OrderNumber)
			http.Error(w, "Invalid order number format", http.StatusUnprocessableEntity)
			return
		}
		err = l.ProcessWithdraw(r.Context(), username, req.OrderNumber, decimal.NewFromFloat(req.Withdrawn))
		if err != nil {
			switch {
			case errors.Is(err, services.ErrInsufficientFunds):
				http.Error(w, "Insufficient funds", http.StatusPaymentRequired)
			default:
				logger.Error("Failed to process withdrawal:", zap.Error(err))
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}

// GetWithdrawHandler — получение информации о выводе средств с накопительного счёта пользователем.
func GetWithdrawHandler(l services.LoyaltyService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// получение данных о пользователе
		username, err := helpers.GetUsername(r.Context())
		if err != nil {
			logger.Warn("Failed to get username:", zap.Error(err))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		withdrawals, err := l.GetWithdrawals(r.Context(), username)
		if err != nil {
			logger.Error("Failed to get user withdrawals:", zap.Error(err))
			http.Error(w, "Server Error", http.StatusInternalServerError)
			return
		}

		if len(withdrawals) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		var response []models.WithdrawalResponse
		for _, w := range withdrawals {
			floatAmount, _ := w.Amount.Float64()
			item := models.WithdrawalResponse{
				Order:       w.OrderNumber,
				Sum:         floatAmount,
				ProcessedAt: w.ProcessedAt.Format(time.RFC3339),
			}
			response = append(response, item)
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			logger.Error("Failed to encode JSON response: ", zap.Error(err))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})
}
