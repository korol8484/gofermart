package order

import (
	"context"
	"errors"
	"github.com/korol8484/gofermart/internal/app/domain"
)

const (
	asciiZero = 48
	asciiTen  = 57
)

type repository interface {
	LoadOrder(ctx context.Context, number string) (*domain.Order, error)
}

type Validate struct {
	nv  domain.OrderNumberValidate
	rep repository
}

func NewValidator(rep repository, nv domain.OrderNumberValidate) *Validate {
	return &Validate{rep: rep, nv: nv}
}

func (v *Validate) Validate(ctx context.Context, number string, userId domain.UserId) ValidateError {
	if err := v.nv.Validate(number); err != nil {
		return err
	}

	order, err := v.rep.LoadOrder(ctx, number)
	if err != nil {
		if errors.Is(err, domain.ErrNotFoundOrder) {
			return nil
		}

		return err
	}

	if order.UserId != userId {
		return ErrorIssetOrderNotOwner
	}

	return ErrorIssetOrder
}
