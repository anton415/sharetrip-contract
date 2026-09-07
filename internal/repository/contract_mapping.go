package repository

import (
	"fmt"

	"github.com/anton415/sharetrip-contract/internal/domain"
	"github.com/anton415/sharetrip-contract/internal/repository/entity"
)

func toDomainContract(row entity.Contract) (domain.Contract, error) {
	id, err := domain.NewContractID(row.ID)
	if err != nil {
		return domain.Contract{}, fmt.Errorf("restore contract id: %w", err)
	}

	clientID, err := domain.NewClientID(row.ClientID)
	if err != nil {
		return domain.Contract{}, fmt.Errorf("restore client id: %w", err)
	}

	status, err := domain.NewContractStatus(row.Status)
	if err != nil {
		return domain.Contract{}, fmt.Errorf("restore contract status: %w", err)
	}

	return domain.RestoreContract(
		id,
		clientID,
		status,
		row.CreatedAt,
		row.UpdatedAt,
		row.ExpiredAt,
	)
}

func toEntityContract(contract domain.Contract) entity.Contract {
	return entity.Contract{
		ID:        contract.ID().Value(),
		ClientID:  contract.ClientID().Value(),
		Status:    string(contract.Status()),
		CreatedAt: contract.CreatedAt(),
		UpdatedAt: contract.UpdatedAt(),
		ExpiredAt: contract.ExpiredAt(),
	}
}
