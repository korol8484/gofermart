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
	Id          int64
	OrderNumber string
	UserId      UserId
	Sum         float64
	Type        BalanceType
	CreatedAt   time.Time
}

type SumBalance struct {
	UserId UserId
	Type   BalanceType
	Sum    float64
}

type SumWC struct {
	Current   float64
	Withdrawn float64
}
