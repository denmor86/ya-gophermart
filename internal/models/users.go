package models

// UserRequest - модель для регистрации и аутентификации пользователя, приходит извне
type UserRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// UserData - модель пользователя из хранищища
type UserData struct {
	UserUUID     string
	Login        string
	PasswordHash string
}
