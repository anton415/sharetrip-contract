package service

import (
	"context"
	"fmt"

	"github.com/anton415/sharetrip-contract/internal/domain"
	"github.com/anton415/sharetrip-contract/internal/repository"
	"github.com/google/uuid"
)

type ContractService struct {
	ServiceCode string
	Enabled     bool
}

type ConfigureContractServicesRequest struct {
	ContractID uuid.UUID
	Services   []ContractService
}

func (s *Service) ConfigureContractServices(ctx context.Context, request ConfigureContractServicesRequest) error {
	id, err := domain.NewContractID(request.ContractID)
	if err != nil {
		return err
	}
	services := make([]domain.ContractService, len(request.Services))
	for index, item := range request.Services {
		services[index] = domain.ContractService{ServiceCode: item.ServiceCode, Enabled: item.Enabled}
	}
	if err := domain.ValidateContractServices(services); err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin configure contract services: %w", err)
	}
	defer rollbackTransaction(tx)

	repo := repository.NewContractRepository(tx)
	contract, err := repo.GetByIDForUpdate(ctx, id)
	if err != nil {
		return err
	}
	if contract.Status() != domain.ContractStatusDraft {
		return domain.ErrContractNotDraft
	}
	if err := repo.ConfigureServices(ctx, id, services); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit configure contract services: %w", err)
	}
	return nil
}
