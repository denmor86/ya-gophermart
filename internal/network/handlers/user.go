package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/denmor86/ya-gophermart/internal/logger"
	"github.com/denmor86/ya-gophermart/internal/models"
	"github.com/denmor86/ya-gophermart/internal/services"
)

// RegisterUserHandler — регистрация нового пользователя
func RegisterUserHandler(i services.IdentityService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// получение данных о пользователе
		var user models.UserRequest
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			logger.Error("Failed to decode request", err)
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		// регистрация в Identity
		if err := i.RegisterUser(r.Context(), user); err != nil {
			// пользователь уже существует
			if errors.Is(err, services.ErrUserAlreadyExists) {
				logger.Warn("Error register user", user.Login)
				http.Error(w, "login already exist", http.StatusConflict)
			} else {
				// ошибка регистрации
				logger.Error("Error register user", err)
				http.Error(w, "Server error", http.StatusInternalServerError)
			}
			return
		}

		// Генерация JWT токена для зарегистрированного пользователя
		token, err := i.GenerateJWT(user.Login)
		if err != nil {
			logger.Error("Failed to generate token", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		// Пользователь зарегистрирован и авторизован
		logger.Info("User registered and authenticated", user.Login)
		w.Header().Set("Authorization", "Bearer "+token)
		w.WriteHeader(http.StatusOK)
	})
}

// AuthenticateUserHandle — аутентификация пользователя
func AuthenticateUserHandle(i services.IdentityService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// получение данных о пользователе
		var user models.UserRequest
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			logger.Error("Failed to decode request", err)
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}
		// аутентификация в Identity
		authenticated, err := i.AuthenticateUser(r.Context(), user)
		if err != nil {
			logger.Error("Error authenticate user", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		// проверка авторизации
		if !authenticated {
			logger.Warn("Authentication failed", user.Login)
			http.Error(w, "Invalid login/password", http.StatusUnauthorized)
			return
		}
		// генерация токена
		token, err := i.GenerateJWT(user.Login)
		if err != nil {
			logger.Error("Failed to generate token", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		// пользователь прошел авторизацию
		logger.Info("User authenticated", user.Login)
		w.Header().Set("Authorization", "Bearer "+token)
		w.WriteHeader(http.StatusOK)
	})
}
