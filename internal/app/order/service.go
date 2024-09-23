package order

import (
	"context"
	"errors"
	"fmt"
	"github.com/korol8484/gofermart/internal/app/domain"
	"go.uber.org/zap"
	"sync"
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

type AccrualResponse struct {
	Order  string
	Status string
	Sum    float64
}

type AccrualRepository interface {
	Process(o domain.Order) (*AccrualResponse, error)
}

type BalanceRepository interface {
	AddBalance(o *domain.Balance) error
}

type ordersRepository interface {
	CreateOrder(ctx context.Context, order *domain.Order) (int64, error)
	LoadOrdersWithBalance(ctx context.Context, userId domain.UserId) ([]domain.OrderWithBalance, error)
	LoadOrdersToProcess(ctx context.Context) ([]domain.Order, error)
	Update(o domain.Order) error
}

type Service struct {
	rep        ordersRepository
	validator  validateOrder
	accrualRep AccrualRepository
	balanceRep BalanceRepository
	log        *zap.Logger

	wg        sync.WaitGroup
	closeChan chan struct{}
}

func NewOrderService(
	rep ordersRepository,
	validator validateOrder,
	accrualRep AccrualRepository,
	balanceRep BalanceRepository,
	log *zap.Logger,
) *Service {
	s := &Service{
		rep:        rep,
		validator:  validator,
		closeChan:  make(chan struct{}),
		balanceRep: balanceRep,
		accrualRep: accrualRep,
		log:        log,
	}

	s.accrual(2 * time.Second)

	return s
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

// accrual -лучше вынести в отдельный сервис
func (s *Service) accrual(d time.Duration) {
	ticker := time.NewTicker(d)

	go func() {
		for {
			select {
			case <-ticker.C:
				s.wg.Add(1)

				err := s.doAccrual()
				if err != nil {
					s.log.With(zap.Error(err)).Error("can't process orders")
				}

				s.wg.Done()
				ticker.Reset(d)
			case <-s.closeChan:
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *Service) doAccrual() error {
	var err error

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orders, err := s.rep.LoadOrdersToProcess(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if len(orders) > 0 {
			for _, o := range orders {
				o.Status = domain.StatusNew
				if err = s.rep.Update(o); err != nil {
					s.log.With(zap.Error(err)).Error("can't update order")
				}
			}
		}
	}()

	for i, o := range orders {
		if err = s.processAccrual(o); err != nil {
			return err
		}

		orders = append(orders[:i], orders[i+1:]...)
	}

	return err
}

func (s *Service) processAccrual(o domain.Order) error {
	var err error
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("unknown panic")
			}
		}
	}()

	res, err := s.accrualRep.Process(o)
	if err != nil {
		return err
	}

	if res.Status == "INVALID" {
		o.Status = domain.StatusInvalid
		if err = s.rep.Update(o); err != nil {
			return err
		}

		return nil
	}

	if res.Status != "PROCESSED" {
		return fmt.Errorf("order in %s", res.Status)
	}

	err = s.balanceRep.AddBalance(&domain.Balance{
		OrderNumber: o.Number,
		UserId:      o.UserId,
		Sum:         res.Sum,
		Type:        domain.BalanceTypeAdd,
		CreatedAt:   time.Now(),
	})
	if err != nil {
		return err
	}

	o.Status = domain.StatusProcessed
	err = s.rep.Update(o)

	return err
}

func (s *Service) Close() {
	close(s.closeChan)

	s.wg.Wait()
}
