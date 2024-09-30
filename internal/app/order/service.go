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
	Validate(ctx context.Context, number string, userID domain.UserID) ValidateError
}

type AccrualResponse struct {
	Order  string
	Status string
	Sum    float64
}

type ErrAccrualRetry struct {
	retry int64
}

func (a *ErrAccrualRetry) Error() string {
	return "Too Many Requests"
}

func (a *ErrAccrualRetry) WithRetryTime(retry int64) {
	a.retry = retry
}

type AccrualClient interface {
	Process(o domain.Order) (*AccrualResponse, error)
}

type BalanceRepository interface {
	AddBalance(o *domain.Balance) error
}

type ordersRepository interface {
	CreateOrder(ctx context.Context, order *domain.Order) (int64, error)
	LoadOrdersWithBalance(ctx context.Context, userID domain.UserID) ([]domain.OrderWithBalance, error)
	LoadOrdersToProcess(ctx context.Context) ([]domain.Order, error)
	Update(o domain.Order) error
}

type Service struct {
	rep          ordersRepository
	validator    validateOrder
	accrualRep   AccrualClient
	balanceRep   BalanceRepository
	log          *zap.Logger
	retryAccrual map[string]time.Time

	mu        sync.RWMutex
	wg        sync.WaitGroup
	closeChan chan struct{}
}

func NewOrderService(
	rep ordersRepository,
	validator validateOrder,
	accrualRep AccrualClient,
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

	s.accrual(1 * time.Second)

	return s
}

func (s *Service) CreateOrder(ctx context.Context, number string, userID domain.UserID) (*domain.Order, error) {
	err := s.validator.Validate(ctx, number, userID)
	if err != nil {
		return nil, err
	}

	order := &domain.Order{
		Number:    number,
		Status:    domain.StatusNew,
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	id, err := s.rep.CreateOrder(ctx, order)
	if err != nil {
		return nil, err
	}

	order.ID = id

	return order, nil
}

func (s *Service) UserOrders(ctx context.Context, userID domain.UserID) ([]domain.OrderWithBalance, error) {
	orders, err := s.rep.LoadOrdersWithBalance(ctx, userID)
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

	if err = s.canProcessAccrual(&o); err != nil {
		return err
	}

	res, err := s.accrualRep.Process(o)
	if err != nil && !errors.Is(err, &ErrAccrualRetry{}) {
		return err
	} else if err != nil && errors.Is(err, &ErrAccrualRetry{}) {
		s.mu.Lock()
		s.retryAccrual[o.Number] = time.Now().Add(time.Duration(err.(*ErrAccrualRetry).retry) * time.Second)
		s.mu.Unlock()

		return err
	}

	s.log.Info(
		"accrualResponse",
		zap.String("status", res.Status),
		zap.Float64("sum", res.Sum),
	)

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
		UserID:      o.UserID,
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

// простой вариант проверки, в реальном сервисе будет оверхед на запросах и проверках,
// такое лучше отправить в отдельную обработку (таску), как вариант добавив в БД поле когда следует запустить
func (s *Service) canProcessAccrual(o *domain.Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if t, ok := s.retryAccrual[o.Number]; ok {
		if time.Now().Before(t) {
			return errors.New("request time has not yet come")
		}

		delete(s.retryAccrual, o.Number)
	}

	return nil
}

func (s *Service) Close() {
	close(s.closeChan)

	s.wg.Wait()
}
