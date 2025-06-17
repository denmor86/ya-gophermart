package helpers

import (
	"context"
	"fmt"

	"github.com/denmor86/ya-gophermart/internal/logger"
	"github.com/go-chi/jwtauth/v5"
)

// GetUsername - извлекает имя пользователя из контекста JWT токена
func GetUsername(context context.Context) (string, error) {
	_, claims, _ := jwtauth.FromContext(context)
	login, ok := claims["username"].(string)
	if !ok {
		logger.Warn("Undefined username from token")
		return "", fmt.Errorf("undefined username")
	}
	return login, nil
}
