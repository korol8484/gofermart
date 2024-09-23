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

type SumWC struct {
	Current   float64
	Withdrawn float64
}
