package services

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/denmor86/ya-gophermart/internal/storage"
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
	GetProcessingOrders(ctx context.Context, count int) ([]string, error)
	ProcessOrder(ctx context.Context, number string) error
}

type Orders struct {
	Storage storage.IStorage
}

// Создание сервиса
func NewOrders(storage storage.IStorage) OrdersService {
	return &Orders{Storage: storage}
}

// AddOrder - добавляет новый заказ, проверяя, не был ли он уже добавлен другим пользователем.
func (s *Orders) AddOrder(ctx context.Context, login string, number string) error {
	// Получаем пользователя по логину
	user, err := s.Storage.GetUser(ctx, login)
	if err != nil {
		return err
	}

	// Проверяем, был ли уже добавлен заказ с таким номером
	existingOrder, err := s.Storage.GetOrder(ctx, number)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return err
	}

	if existingOrder != nil {
		// Если заказ добавлен текущим пользователем
		if existingOrder.UserUUID == user.UserUUID {
			return ErrOrderAlreadyUploaded
		}
		// Если заказ добавлен другим пользователем
		return ErrOrderUploadedByAnother
	}

	// Добавление заказа
	err = s.Storage.AddOrder(ctx, number, user.UserUUID, time.Now())
	if err != nil {
		return err
	}

	return nil
}
func (s *Orders) GetProcessingOrders(ctx context.Context, count int) ([]string, error) {
	return s.Storage.GetProcessingOrders(ctx, count)
}

// ProcessOrder - обработка заказа, запрос начисления вознаграждений
func (s *Orders) ProcessOrder(ctx context.Context, number string) error {

	return nil
}

// CheckNumber проверяет строку используя алгоритм Луна
func CheckNumber(number string) bool {
	// Удаляем все пробелы
	number = strings.ReplaceAll(number, " ", "")

	// Проверяем, что строка состоит только из цифр
	if _, err := strconv.Atoi(number); err != nil {
		return false
	}

	sum := 0
	alternate := false

	// Идем по цифрам справа налево
	for i := len(number) - 1; i >= 0; i-- {
		digit, _ := strconv.Atoi(string(number[i]))

		if alternate {
			digit *= 2
			if digit > 9 {
				digit = (digit % 10) + 1
			}
		}

		sum += digit
		alternate = !alternate
	}

	// Число валидно, если сумма кратна 10
	return sum%10 == 0
}
