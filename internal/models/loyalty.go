package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// WithdrawalRequest - модель запроса списания баллов за заказ
type WithdrawalRequest struct {
	OrderNumber string  `json:"order"`
	Withdrawn   float64 `json:"sum"`
}

// WithdrawalData - модель хранения данных по начислениям баллов по заказам
type WithdrawalData struct {
	OrderNumber string
	UserID      string
	Amount      decimal.Decimal
	ProcessedAt time.Time
}

// WithdrawalResponse — структура ответа о выводе средств
type WithdrawalResponse struct {
	Order       string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}
