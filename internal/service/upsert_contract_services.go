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

type UpsertContractServicesRequest struct {
	ContractID uuid.UUID
	Services   []ContractService
}

func (s *Service) UpsertContractServices(ctx context.Context, request UpsertContractServicesRequest) error {
	id, err := domain.NewContractID(request.ContractID)
	if err != nil {
		return err
	}
	services := make([]domain.ContractService, len(request.Services))
	for i, item := range request.Services {
		services[i] = domain.ContractService{ServiceCode: item.ServiceCode, Enabled: item.Enabled}
	}
	if err := domain.ValidateContractServices(services); err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin upsert contract services: %w", err)
	}
	defer rollbackTransaction(tx)

	repo := repository.NewContractRepository(tx)
	if _, err := repo.GetByIDForUpdate(ctx, id); err != nil {
		return err
	}
	if err := repo.UpsertServices(ctx, id, services); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit upsert contract services: %w", err)
	}
	return nil
}
