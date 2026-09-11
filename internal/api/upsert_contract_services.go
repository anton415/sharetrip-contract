package api

import (
	"github.com/anton415/sharetrip-contract/gen"
	"github.com/anton415/sharetrip-contract/internal/domain"
	"github.com/anton415/sharetrip-contract/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (s *Server) UpsertContractServices(ctx *fiber.Ctx, contractID string) error {
	id, err := uuid.Parse(contractID)
	if err != nil || id == uuid.Nil {
		return writeError(ctx, domain.ErrInvalidContractID)
	}
	var request gen.UpsertContractServicesRequest
	if err := parseRequest(ctx, &request); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(gen.ErrorResponse{Error: "invalid request body"})
	}
	services := make([]service.ContractService, len(request.Services))
	for i, item := range request.Services {
		if item.Enabled == nil {
			return writeError(ctx, domain.ErrInvalidContractServices)
		}
		services[i] = service.ContractService{ServiceCode: string(item.ServiceCode), Enabled: *item.Enabled}
	}
	if err := s.contractService.UpsertContractServices(ctx.UserContext(), service.UpsertContractServicesRequest{
		ContractID: id,
		Services:   services,
	}); err != nil {
		return writeError(ctx, err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}
