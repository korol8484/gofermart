package domain

import (
	"errors"
	"time"
)

type Status string

var (
	StatusNew        Status = "NEW"
	StatusProcessing Status = "PROCESSING"
	StatusInvalid    Status = "INVALID"
	StatusProcessed  Status = "PROCESSED"
)

var (
	ErrNotFoundOrder = errors.New("order not found")
)

type Order struct {
	ID        int64
	Number    string
	Status    Status
	UserID    UserID
	CreatedAt time.Time
}

type NumberValidateError error

var (
	ErrorNumberValidateFormat NumberValidateError = errors.New("неверный формат номера заказа")
)

type OrderNumberValidate interface {
	Validate(number string) NumberValidateError
}
