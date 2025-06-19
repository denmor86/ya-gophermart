package models

import "github.com/shopspring/decimal"

// UserRequest - модель для регистрации и аутентификации пользователя, приходит извне
type UserRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// UserData - модель пользователя из хранищища
type UserData struct {
	UserID       string
	Login        string
	PasswordHash string
	Balance      decimal.Decimal
}

// UserBalance - модель баланса пользователя
type UserBalance struct {
	Current   float64 `json:"current"`   // Текущий баланс пользователя
	Withdrawn float64 `json:"withdrawn"` // Общая сумма выведенных средств
}
