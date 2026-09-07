package domain

import (
	"time"

	"github.com/google/uuid"
)

type CreateContractRequest struct {
	ClientID  ClientID
	ExpiredAt *time.Time
}

type CreateContractResponse struct {
	ID        ContractID
	ClientID  ClientID
	Status    ContractStatus
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiredAt *time.Time
}

func CreateContract(
	request CreateContractRequest,
) (CreateContractResponse, error) {
	if request.ClientID.Value() == uuid.Nil {
		return CreateContractResponse{}, ErrInvalidClientID
	}

	id, err := NewContractID(uuid.New())
	if err != nil {
		return CreateContractResponse{}, err
	}

	now := time.Now().UTC()

	var expiredAt *time.Time
	if request.ExpiredAt != nil {
		value := *request.ExpiredAt
		expiredAt = &value
	}

	contract := Contract{
		id:        id,
		clientID:  request.ClientID,
		status:    ContractStatusDraft,
		createdAt: now,
		updatedAt: now,
		expiredAt: expiredAt,
	}

	return CreateContractResponse{
		ID:        contract.ID(),
		ClientID:  contract.ClientID(),
		Status:    contract.Status(),
		CreatedAt: contract.CreatedAt(),
		UpdatedAt: contract.UpdatedAt(),
		ExpiredAt: contract.ExpiredAt(),
	}, nil
}
