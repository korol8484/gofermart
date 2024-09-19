package domain

import "time"

type Status string

var (
	StatusNew        Status = "NEW"
	StatusProcessing Status = "PROCESSING"
	StatusInvalid    Status = "INVALID"
	StatusProcessed  Status = "PROCESSED"
)

type Order struct {
	Id        int64
	Number    string
	Status    Status
	UserId    UserId
	CreatedAt time.Time
}
