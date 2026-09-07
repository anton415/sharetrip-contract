package entity

import (
	"time"

	"github.com/google/uuid"
)

type Contract struct {
	ID        uuid.UUID
	ClientID  uuid.UUID
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiredAt *time.Time
}
