package models

// User - модель пользователя для регистрацию и аутентификации пользователя
type User struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
