package service

import (
	"context"
	"fmt"
	"time"

	"github.com/anton415/sharetrip-contract/internal/domain"
	"github.com/anton415/sharetrip-contract/internal/repository"
	"github.com/google/uuid"
)

type GetContractRequest struct {
	ContractID uuid.UUID
}

type GetActiveContractRequest struct {
	ClientID uuid.UUID
}

type GetContractResponse struct {
	ID        uuid.UUID
	ClientID  uuid.UUID
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiredAt *time.Time
}

func (s *Service) GetContract(ctx context.Context, request GetContractRequest) (GetContractResponse, error) {
	id, err := domain.NewContractID(request.ContractID)
	if err != nil {
		return GetContractResponse{}, err
	}
	return s.getContract(ctx, func(repo *repository.ContractRepository) (domain.Contract, error) {
		return repo.GetByID(ctx, id)
	})
}

func (s *Service) GetActiveContract(ctx context.Context, request GetActiveContractRequest) (GetContractResponse, error) {
	id, err := domain.NewClientID(request.ClientID)
	if err != nil {
		return GetContractResponse{}, err
	}
	return s.getContract(ctx, func(repo *repository.ContractRepository) (domain.Contract, error) {
		return repo.GetActiveByClientID(ctx, id)
	})
}

func (s *Service) getContract(
	ctx context.Context,
	load func(*repository.ContractRepository) (domain.Contract, error),
) (GetContractResponse, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return GetContractResponse{}, fmt.Errorf("begin get contract: %w", err)
	}
	defer rollbackTransaction(tx)

	contract, err := load(repository.NewContractRepository(tx))
	if err != nil {
		return GetContractResponse{}, err
	}
	return GetContractResponse{
		ID:        contract.ID().Value(),
		ClientID:  contract.ClientID().Value(),
		Status:    string(contract.Status()),
		CreatedAt: contract.CreatedAt(),
		UpdatedAt: contract.UpdatedAt(),
		ExpiredAt: contract.ExpiredAt(),
	}, nil
}
