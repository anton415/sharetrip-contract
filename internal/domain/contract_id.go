package domain

import "github.com/google/uuid"

type ContractID struct {
	value uuid.UUID
}

func NewContractID(value uuid.UUID) (ContractID, error) {
	if value == uuid.Nil {
		return ContractID{}, ErrInvalidContractID
	}

	return ContractID{value: value}, nil
}

func (id ContractID) Value() uuid.UUID {
	return id.value
}
