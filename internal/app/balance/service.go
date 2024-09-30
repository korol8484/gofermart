package balance

import (
	"context"
	"github.com/korol8484/gofermart/internal/app/domain"
)

type repository interface {
	GetUserWithdrawals(ctx context.Context, userID domain.UserID) ([]*domain.Balance, error)
	GetUserSum(ctx context.Context, userID domain.UserID, types ...domain.BalanceType) ([]*domain.SumBalance, error)
	Withdraw(ctx context.Context, userID domain.UserID, number string, sum float64) (*domain.Balance, error)
}

type Service struct {
	nv  domain.OrderNumberValidate
	rep repository
}

func NewBalanceService(rep repository, nv domain.OrderNumberValidate) *Service {
	return &Service{
		nv:  nv,
		rep: rep,
	}
}

func (s *Service) LoadWithdrawals(ctx context.Context, userID domain.UserID) ([]*domain.Balance, error) {
	withdrawals, err := s.rep.GetUserWithdrawals(ctx, userID)
	if err != nil {
		return nil, err
	}

	return withdrawals, nil
}

func (s *Service) LoadSum(ctx context.Context, userID domain.UserID) (*domain.Sum, error) {
	sums, err := s.rep.GetUserSum(ctx, userID, domain.BalanceTypeAdd, domain.BalanceTypeWithdrawn)
	if err != nil {
		return nil, err
	}

	sumWC := &domain.Sum{}
	for _, wc := range sums {
		switch wc.Type {
		case domain.BalanceTypeAdd:
			sumWC.Current = wc.Sum
		case domain.BalanceTypeWithdrawn:
			sumWC.Withdrawn = wc.Sum
		}
	}

	if sumWC.Current > 0 {
		sumWC.Current = sumWC.Current - sumWC.Withdrawn
	}

	return sumWC, nil
}

func (s *Service) Withdraw(ctx context.Context, userID domain.UserID, number string, sum float64) (*domain.Balance, error) {
	if err := s.nv.Validate(number); err != nil {
		return nil, err
	}

	return s.rep.Withdraw(ctx, userID, number, sum)
}
