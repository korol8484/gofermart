package balance

import (
	"context"
	"github.com/korol8484/gofermart/internal/app/domain"
)

type repository interface {
	GetUserWithdrawals(ctx context.Context, userId domain.UserID) ([]*domain.Balance, error)
	GetUserSum(ctx context.Context, userId domain.UserID, types ...domain.BalanceType) ([]*domain.SumBalance, error)
	Withdraw(ctx context.Context, userId domain.UserID, number string, sum float64) (*domain.Balance, error)
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

func (s *Service) UserWithdrawals(ctx context.Context, userId domain.UserID) ([]*domain.Balance, error) {
	withdrawals, err := s.rep.GetUserWithdrawals(ctx, userId)
	if err != nil {
		return nil, err
	}

	return withdrawals, nil
}

func (s *Service) GetUserSumWC(ctx context.Context, userId domain.UserID) (*domain.SumWC, error) {
	sums, err := s.rep.GetUserSum(ctx, userId, domain.BalanceTypeAdd, domain.BalanceTypeWithdrawn)
	if err != nil {
		return nil, err
	}

	sumWC := &domain.SumWC{}
	for _, wc := range sums {
		switch wc.Type {
		case domain.BalanceTypeAdd:
			sumWC.Current = wc.Sum
		case domain.BalanceTypeWithdrawn:
			sumWC.Withdrawn = wc.Sum
		}
	}

	return sumWC, nil
}

func (s *Service) Withdraw(ctx context.Context, userId domain.UserID, number string, sum float64) (*domain.Balance, error) {
	if err := s.nv.Validate(number); err != nil {
		return nil, err
	}

	return s.rep.Withdraw(ctx, userId, number, sum)
}
