package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// Статусы заказов
const (
	OrderStatusInvalid    = "INVALID"
	OrderStatusNew        = "NEW"
	OrderStatusProcessed  = "PROCESSED"
	OrderStatusProcessing = "PROCESSING"
	OrderStatusRegistered = "REGISTERED"
)

// OrderResponse - модель заказа пользователя для выдачи
type OrderResponse struct {
	Number     string  `json:"number"`
	Status     string  `json:"status"`
	Accrual    float64 `json:"accrual,omitempty"`
	UploadedAt string  `json:"uploaded_at"`
}

// Order - модель заказа пользователя
type OrderData struct {
	Number     string
	UserID     string
	Status     string
	Accrual    decimal.Decimal
	UploadedAt time.Time
}
