package services

import (
	"context"
	"errors"
	"time"

	"github.com/denmor86/ya-gophermart/internal/config"
	"github.com/denmor86/ya-gophermart/internal/logger"
	"github.com/denmor86/ya-gophermart/internal/models"
	"github.com/denmor86/ya-gophermart/internal/storage"
	"github.com/go-chi/jwtauth"
	"golang.org/x/crypto/bcrypt"
)

type Identity struct {
	JWTAuth *jwtauth.JWTAuth
	Storage *storage.Database
}

var (
	ErrUserAlreadyExists = errors.New("user already exists")
)

const (
	TokenSecterAlgo     = "HS256"
	TokenExpirationTime = 24 * time.Hour
)

// Создание сервиса
func NewIdentity(cfg config.Config, storage *storage.Database) *Identity {
	tokenAuth := jwtauth.New(TokenSecterAlgo, []byte(cfg.JWTSecret), nil)
	return &Identity{JWTAuth: tokenAuth, Storage: storage}
}

// Регистрация нового пользователя.
func (i *Identity) RegisterUser(context context.Context, user models.User) error {
	logger.Info("Register user:", user.Login)

	userUUID, _ := i.Storage.GetUserUUID(context, user.Login)
	if userUUID != "" {
		logger.Warn("User already exist")
		return ErrUserAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("Error generating password hash", err)
		return err
	}

	err = i.Storage.AddUser(context, user.Login, string(hashedPassword))
	if err != nil {
		logger.Error("Error registering user", user.Login, err)
		return err
	}
	return nil
}

// Аутентификация пользователя
func (s *Identity) AuthenticateUser(context context.Context, user models.User) (bool, error) {
	logger.Info("Authenticate user", user.Login)

	password, err := s.Storage.GetUserPassword(context, user.Login)
	if err != nil {
		logger.Error("Error getting user password", err)
		return false, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(password), []byte(user.Password))
	if err != nil {
		logger.Warn("Invalid password", user.Login)
		return false, nil
	}

	logger.Info("User authenticated", user.Login)
	return true, nil
}

// Создание строки JWT токена
func (i *Identity) GenerateJWT(username string) (string, error) {
	expirationTime := time.Now().Add(TokenExpirationTime)

	_, tokenString, err := i.JWTAuth.Encode(map[string]interface{}{
		"username": username,
		"exp":      expirationTime,
	})
	return tokenString, err
}

// Возвращаем указатель на JWTAuth (chi)
func (i *Identity) GetTokenAuth() *jwtauth.JWTAuth {
	return i.JWTAuth
}
