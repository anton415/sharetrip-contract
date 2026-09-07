package domain

import "github.com/google/uuid"

type ClientID struct {
	value uuid.UUID
}

func NewClientID(value uuid.UUID) (ClientID, error) {
	if value == uuid.Nil {
		return ClientID{}, ErrInvalidClientID
	}

	return ClientID{value: value}, nil
}

func (id ClientID) Value() uuid.UUID {
	return id.value
}
