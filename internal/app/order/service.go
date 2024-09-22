package order

import (
	"context"
	"errors"
	"github.com/korol8484/gofermart/internal/app/domain"
	"time"
)

type ValidateError error

var (
	ErrorIssetOrder         ValidateError = errors.New("номер заказа уже был загружен этим пользователем")
	ErrorIssetOrderNotOwner ValidateError = errors.New("номер заказа уже был загружен другим пользователем")
	ErrorInvalidFormat      ValidateError = domain.ErrorNumberValidateFormat
)

type validateOrder interface {
	Validate(ctx context.Context, number string, userId domain.UserId) ValidateError
}

type ordersRepository interface {
	CreateOrder(ctx context.Context, order *domain.Order) (int64, error)
	LoadOrdersWithBalance(ctx context.Context, userId domain.UserId) ([]domain.OrderWithBalance, error)
}

type Service struct {
	rep       ordersRepository
	validator validateOrder
}

func NewOrderService(rep ordersRepository, validator validateOrder) *Service {
	return &Service{
		rep:       rep,
		validator: validator,
	}
}

func (s *Service) CreateOrder(ctx context.Context, number string, userId domain.UserId) (*domain.Order, error) {
	err := s.validator.Validate(ctx, number, userId)
	if err != nil {
		return nil, err
	}

	order := &domain.Order{
		Number:    number,
		Status:    domain.StatusNew,
		UserId:    userId,
		CreatedAt: time.Now(),
	}

	id, err := s.rep.CreateOrder(ctx, order)
	if err != nil {
		return nil, err
	}

	order.Id = id

	return order, nil
}

func (s *Service) UserOrders(ctx context.Context, userId domain.UserId) ([]domain.OrderWithBalance, error) {
	orders, err := s.rep.LoadOrdersWithBalance(ctx, userId)
	if err != nil {
		return nil, err
	}

	return orders, nil
}
