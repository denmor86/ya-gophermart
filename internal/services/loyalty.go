package services

import (
	"context"
	"errors"

	"github.com/denmor86/ya-gophermart/internal/logger"
	"github.com/denmor86/ya-gophermart/internal/models"
	"github.com/denmor86/ya-gophermart/internal/storage"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

var (
	ErrInsufficientFunds       = errors.New("insufficient funds for withdrawal")
	ErrInvalidWithdrawalAmount = errors.New("invalid withdrawal amount")
)

type LoyaltyService interface {
	GetBalance(ctx context.Context, login string) (*models.UserBalance, error)
	GetWithdrawals(ctx context.Context, login string) ([]models.WithdrawalData, error)
	ProcessWithdraw(ctx context.Context, login string, order string, sum decimal.Decimal) error
}

type Loyalty struct {
	Storage storage.IStorage
}

// Создание сервиса
func NewLoyalty(storage storage.IStorage) LoyaltyService {
	return &Loyalty{Storage: storage}
}

// GetBalance возващает баланс баллов пользователя
func (s *Loyalty) GetBalance(ctx context.Context, login string) (*models.UserBalance, error) {
	// Получаем баланс пользователя и сумму снятых средств из хранилища
	userBalance, err := s.Storage.GetUserBalance(ctx, login)
	if err != nil {
		logger.Error("Failed to get user balance", zap.Error(err))
		return nil, err
	}

	return userBalance, nil
}

// GetLoyalty возвращает список всех выводов средств пользователя по его логину
func (s *Loyalty) GetWithdrawals(ctx context.Context, login string) ([]models.WithdrawalData, error) {
	user, err := s.Storage.GetUser(ctx, login)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			logger.Warn("User not found", login)
			return nil, storage.ErrUserNotFound
		}
		logger.Error("Error getting user", zap.Error(err))
		return nil, err
	}

	// Получаем список всех выводов средств пользователя
	withdrawals, err := s.Storage.GetWithdrawals(ctx, user.UserID)
	if err != nil {
		logger.Error("Failed to get withdrawals:", zap.Error(err))
		return nil, err
	}

	return withdrawals, nil
}

// ProcessWithdraw обработка запроса вывода средств для пользователя и заказа
func (s *Loyalty) ProcessWithdraw(ctx context.Context, login string, orderNumber string, sum decimal.Decimal) error {
	user, err := s.Storage.GetUser(ctx, login)
	if err != nil {
		logger.Error("Failed to get user", zap.Error(err))
		return err
	}

	// Проверка на отрицательную сумму при выводе средств
	if sum.LessThan(decimal.Zero) {
		return ErrInvalidWithdrawalAmount
	}

	// Проверяем, достаточно ли средств для вывода
	if user.Balance.LessThan(sum) {
		return ErrInsufficientFunds
	}

	withdrawal := models.WithdrawalData{
		OrderNumber: orderNumber,
		UserID:      user.UserID,
		Amount:      sum,
	}

	// Добавляем информацию о выводе и обновляем баланс пользователя
	return s.Storage.AddWithdrawal(ctx, withdrawal)
}
