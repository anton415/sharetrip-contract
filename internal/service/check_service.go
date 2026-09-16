package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anton415/sharetrip-contract/internal/domain"
	"github.com/anton415/sharetrip-contract/internal/repository"
	"github.com/google/uuid"
)

type CheckServiceRequest struct {
	ClientID    uuid.UUID
	ServiceCode string
}

type CheckServiceResponse struct {
	Allowed bool
	Reason  string
}

func (s *Service) CheckService(ctx context.Context, request CheckServiceRequest) (CheckServiceResponse, error) {
	if request.ClientID == uuid.Nil {
		return CheckServiceResponse{}, domain.ErrInvalidClientID
	}
	if request.ServiceCode == "" {
		return CheckServiceResponse{}, domain.ErrInvalidContractServices
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CheckServiceResponse{}, fmt.Errorf("begin check service: %w", err)
	}
	defer rollbackTransaction(tx)

	contract, enabled, err := repository.NewContractRepository(tx).GetForServiceCheck(ctx, request.ClientID, request.ServiceCode)
	if errors.Is(err, repository.ErrContractNotFound) {
		return CheckServiceResponse{Allowed: false, Reason: "contract_not_found"}, nil
	}
	if err != nil {
		return CheckServiceResponse{}, err
	}
	allowed, reason := contract.CheckService(enabled, time.Now())
	return CheckServiceResponse{Allowed: allowed, Reason: reason}, nil
}
