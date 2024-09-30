package domain

import (
	"errors"
	"time"
)

type BalanceType int

const (
	BalanceTypeAdd BalanceType = iota
	BalanceTypeWithdrawn
)

var (
	ErrBalanceInsufficientFunds = errors.New("на счету недостаточно средств")
)

type Balance struct {
	ID          int64
	OrderNumber string
	UserID      UserID
	Sum         float64
	Type        BalanceType
	CreatedAt   time.Time
}

type SumBalance struct {
	UserID UserID
	Type   BalanceType
	Sum    float64
}

type Sum struct {
	Current   float64
	Withdrawn float64
}

func ConvertFromCurrencyUnit(s int64) float64 {
	return float64(s) / 100
}

func ConvertToCurrencyUnit(s float64) int64 {
	return int64(s * 100)
}
