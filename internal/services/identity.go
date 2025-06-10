package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/denmor86/ya-gophermart/internal/logger"
	"github.com/denmor86/ya-gophermart/internal/models"
	"github.com/denmor86/ya-gophermart/internal/storage"
	"github.com/go-chi/jwtauth/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserAlreadyExists = errors.New("user already exists")
)

const (
	TokenSecterAlgo     = "HS256"
	TokenExpirationTime = 24 * time.Hour
)

type IdentityService interface {
	RegisterUser(context context.Context, user models.UserRequest) error
	AuthenticateUser(context context.Context, user models.UserRequest) (bool, error)
	GenerateJWT(username string) (string, error)
	GetTokenAuth() *jwtauth.JWTAuth
}

type Identity struct {
	JWTAuth *jwtauth.JWTAuth
	Storage storage.IStorage
}

// Создание сервиса
func NewIdentity(JWTSecret string, storage storage.IStorage) IdentityService {
	tokenAuth := jwtauth.New(TokenSecterAlgo, []byte(JWTSecret), nil)
	return &Identity{JWTAuth: tokenAuth, Storage: storage}
}

// Регистрация нового пользователя.
func (i *Identity) RegisterUser(context context.Context, user models.UserRequest) error {
	logger.Info("Register user:", user.Login)

	userData, _ := i.Storage.GetUser(context, user.Login)
	if userData != nil {
		logger.Warn("User already exist")
		return ErrUserAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("Error generating password hash:", zap.Error(err))
		return err
	}

	err = i.Storage.AddUser(context, user.Login, string(hashedPassword))
	if err != nil {
		logger.Error("Error registering user", user.Login, zap.Error(err))
		return err
	}
	return nil
}

// Аутентификация пользователя
func (s *Identity) AuthenticateUser(context context.Context, user models.UserRequest) (bool, error) {
	logger.Info("Authenticate user", user.Login)

	userData, err := s.Storage.GetUser(context, user.Login)
	if err != nil {
		logger.Error("Error getting user password:", zap.Error(err))
		return false, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(userData.PasswordHash), []byte(user.Password))
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

// GetUsername - извлекает имя пользователя из контекста JWT токена
func GetUsername(context context.Context) (string, error) {
	_, claims, _ := jwtauth.FromContext(context)
	login, ok := claims["username"].(string)
	if !ok {
		logger.Warn("undefined username from token")
		return "", fmt.Errorf("undefined username")
	}
	return login, nil
}
