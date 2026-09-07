package service

import (
	"context"
	"fmt"
	"time"

	"github.com/anton415/sharetrip-contract/internal/domain"
	"github.com/anton415/sharetrip-contract/internal/repository"
	"github.com/google/uuid"
)

type SignContractRequest struct {
	ContractID uuid.UUID
}

type SignContractResponse struct {
	ID        uuid.UUID
	Status    string
	UpdatedAt time.Time
}

func (s *Service) SignContract(
	ctx context.Context,
	request SignContractRequest,
) (SignContractResponse, error) {
	contractID, err := domain.NewContractID(request.ContractID)
	if err != nil {
		return SignContractResponse{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SignContractResponse{}, fmt.Errorf("begin sign contract: %w", err)
	}
	defer rollbackTransaction(tx)

	repo := repository.NewContractRepository(tx)
	contract, err := repo.GetByIDForUpdate(ctx, contractID)
	if err != nil {
		return SignContractResponse{}, fmt.Errorf(
			"get contract by id: %w", err,
		)
	}

	signed, err := domain.SignContract(domain.SignContractRequest{
		Contract: contract,
	})
	if err != nil {
		return SignContractResponse{}, fmt.Errorf(
			"sign contract: %w", err,
		)
	}

	if err := repo.Update(ctx, signed.Contract); err != nil {
		return SignContractResponse{}, fmt.Errorf(
			"update contract: %w", err,
		)
	}
	if err := tx.Commit(ctx); err != nil {
		return SignContractResponse{}, fmt.Errorf("commit sign contract: %w", err)
	}

	return SignContractResponse{
		ID:        signed.Contract.ID().Value(),
		Status:    string(signed.Contract.Status()),
		UpdatedAt: signed.Contract.UpdatedAt(),
	}, nil
}
