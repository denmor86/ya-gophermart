package services

import (
	"context"
	"errors"
	"time"

	"github.com/denmor86/ya-gophermart/internal/client"
	"github.com/denmor86/ya-gophermart/internal/logger"
	"github.com/denmor86/ya-gophermart/internal/models"
	"github.com/denmor86/ya-gophermart/internal/storage"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

var (
	ErrOrderAlreadyExists     = errors.New("order already exists")
	ErrOrderAlreadyUploaded   = errors.New("order already uploaded by this user")
	ErrOrderNotFound          = errors.New("order not found")
	ErrOrderUploadedByAnother = errors.New("order already uploaded by another user")
)

// OrdersService - представляет интерфейс для работы с сервисом заказов
type OrdersService interface {
	AddOrder(ctx context.Context, login string, number string) error
	GetOrders(ctx context.Context, login string) ([]models.OrderData, error)
	ClaimOrdersForProcessing(ctx context.Context, count int) ([]string, error)
	ProcessOrder(ctx context.Context, number string) error
}

type Orders struct {
	OrdersStorage storage.OrdersStorage
	UsersStorage  storage.UsersStorage
	Accrual       client.AccrualService
}

// Создание сервиса
func NewOrders(accrual client.AccrualService, orders storage.OrdersStorage, users storage.UsersStorage) OrdersService {
	return &Orders{OrdersStorage: orders, UsersStorage: users, Accrual: accrual}
}

// AddOrder - добавляет новый заказ, проверяя, не был ли он уже добавлен другим пользователем.
func (s *Orders) AddOrder(ctx context.Context, login string, number string) error {
	// Получаем пользователя по логину
	user, err := s.UsersStorage.GetUser(ctx, login)
	if err != nil {
		return err
	}

	// Проверяем, был ли уже добавлен заказ с таким номером
	existingOrder, err := s.OrdersStorage.GetOrder(ctx, number)
	if err != nil && !errors.Is(err, storage.ErrOrderNotFound) {
		logger.Warn("failed to add order:", zap.Error(err))
		return err
	}

	if existingOrder != nil {
		// Если заказ добавлен текущим пользователем
		if existingOrder.UserID == user.UserID {
			return ErrOrderAlreadyUploaded
		}
		// Если заказ добавлен другим пользователем
		return ErrOrderUploadedByAnother
	}

	// Добавление заказа
	err = s.OrdersStorage.AddOrder(ctx, number, user.UserID, time.Now())
	if err != nil {
		return err
	}

	return nil
}

// GetOrders - возвращает список заказов пользователя.
func (s *Orders) GetOrders(ctx context.Context, login string) ([]models.OrderData, error) {
	user, err := s.UsersStorage.GetUser(ctx, login)
	if err != nil {
		return nil, err
	}

	orders, err := s.OrdersStorage.GetOrders(ctx, user.UserID)
	if err != nil {
		return nil, err
	}

	return orders, nil
}

// ClaimOrdersForProcessing - сформировать список номеров заказов из находящихся в статусе 'NEW' и установить им статус 'PROCESSING'
func (s *Orders) ClaimOrdersForProcessing(ctx context.Context, count int) ([]string, error) {
	return s.OrdersStorage.ClaimOrdersForProcessing(ctx, count)
}

// ProcessOrder - обработка заказа, запрос начисления вознаграждений
func (s *Orders) ProcessOrder(ctx context.Context, number string) error {
	accrual, status, err := s.Accrual.GetOrderAccrual(ctx, number)
	if err != nil {
		logger.Warn("Failed to get order", number, "accrual. Error:", zap.Error(err))
		// оставляем статус PROCESSING, изменяем количество попыток запросов
		status = models.OrderStatusProcessing
	}
	// устанавливаем статус, количество баллов и обновляем баланс баллов пользователя
	return s.OrdersStorage.UpdateOrderAndBalance(ctx, number, status, decimal.NewFromFloat(accrual))
}
