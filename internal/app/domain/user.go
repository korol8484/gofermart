package domain

import "errors"

var (
	ErrIssetUser = errors.New("логин уже занят")
)

type UserId int64

type User struct {
	ID           UserId
	Login        string
	PasswordHash string
}
