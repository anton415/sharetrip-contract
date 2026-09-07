package service

import (
	"context"
	"fmt"
	"time"

	"github.com/anton415/sharetrip-contract/internal/domain"
	"github.com/anton415/sharetrip-contract/internal/repository"
	"github.com/google/uuid"
)

type CreateContractRequest struct {
	ClientID  uuid.UUID
	ExpiredAt *time.Time
}

type CreateContractResponse struct {
	ID        uuid.UUID
	ClientID  uuid.UUID
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiredAt *time.Time
}

func (s *Service) CreateContract(
	ctx context.Context,
	request CreateContractRequest,
) (CreateContractResponse, error) {
	clientID, err := domain.NewClientID(request.ClientID)
	if err != nil {
		return CreateContractResponse{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CreateContractResponse{}, fmt.Errorf("begin create contract: %w", err)
	}
	defer rollbackTransaction(tx)

	created, err := domain.CreateContract(domain.CreateContractRequest{
		ClientID:  clientID,
		ExpiredAt: request.ExpiredAt,
	})
	if err != nil {
		return CreateContractResponse{}, err
	}

	contract, err := domain.RestoreContract(
		created.ID,
		created.ClientID,
		created.Status,
		created.CreatedAt,
		created.UpdatedAt,
		created.ExpiredAt,
	)
	if err != nil {
		return CreateContractResponse{}, fmt.Errorf(
			"build created contract: %w", err,
		)
	}

	repo := repository.NewContractRepository(tx)

	if err := repo.Create(ctx, contract); err != nil {
		return CreateContractResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CreateContractResponse{}, fmt.Errorf("commit create contract: %w", err)
	}

	return CreateContractResponse{
		ID:        contract.ID().Value(),
		ClientID:  contract.ClientID().Value(),
		Status:    string(contract.Status()),
		CreatedAt: contract.CreatedAt(),
		UpdatedAt: contract.UpdatedAt(),
		ExpiredAt: contract.ExpiredAt(),
	}, nil
}
