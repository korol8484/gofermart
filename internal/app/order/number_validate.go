package order

import (
	"errors"
	"github.com/korol8484/gofermart/internal/app/domain"
)

type NumberValidate struct{}

func NewNumberValidate() *NumberValidate {
	return &NumberValidate{}
}

func (n *NumberValidate) Validate(number string) domain.NumberValidateError {
	if err := n.validateLun(number); err != nil {
		return domain.ErrorNumberValidateFormat
	}

	return nil
}

func (n *NumberValidate) validateLun(number string) error {
	p := len(number) % 2
	sum, err := n.calculateLuhnSum(number, p)
	if err != nil {
		return err
	}

	if sum%10 != 0 {
		return errors.New("invalid number")
	}

	return nil
}

func (n *NumberValidate) calculateLuhnSum(number string, parity int) (int64, error) {
	var sum int64
	for i, d := range number {
		if d < asciiZero || d > asciiTen {
			return 0, errors.New("invalid digit")
		}

		d = d - asciiZero
		if i%2 == parity {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}

		sum += int64(d)
	}

	return sum, nil
}
