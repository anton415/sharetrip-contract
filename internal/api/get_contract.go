package api

import (
	"github.com/anton415/sharetrip-contract/gen"
	"github.com/anton415/sharetrip-contract/internal/domain"
	"github.com/anton415/sharetrip-contract/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (s *Server) GetContract(ctx *fiber.Ctx, contractID string) error {
	id, err := uuid.Parse(contractID)
	if err != nil || id == uuid.Nil {
		return writeError(ctx, domain.ErrInvalidContractID)
	}
	response, err := s.contractService.GetContract(ctx.UserContext(), service.GetContractRequest{ContractID: id})
	if err != nil {
		return writeError(ctx, err)
	}
	return writeContract(ctx, response)
}

func (s *Server) GetActiveContract(ctx *fiber.Ctx, clientID string) error {
	id, err := uuid.Parse(clientID)
	if err != nil || id == uuid.Nil {
		return writeError(ctx, domain.ErrInvalidClientID)
	}
	response, err := s.contractService.GetActiveContract(ctx.UserContext(), service.GetActiveContractRequest{ClientID: id})
	if err != nil {
		return writeError(ctx, err)
	}
	return writeContract(ctx, response)
}

func writeContract(ctx *fiber.Ctx, response service.GetContractResponse) error {
	status, err := toAPIContractStatus(response.Status)
	if err != nil {
		return writeError(ctx, err)
	}
	return ctx.Status(fiber.StatusOK).JSON(gen.GetContractResponse{
		Contract: gen.Contract{
			Id:        response.ID,
			ClientId:  response.ClientID,
			Status:    status,
			CreatedAt: response.CreatedAt,
			UpdatedAt: response.UpdatedAt,
			ExpiredAt: response.ExpiredAt,
		},
	})
}
