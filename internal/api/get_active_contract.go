package api

import (
	"github.com/anton415/sharetrip-contract/internal/domain"
	"github.com/anton415/sharetrip-contract/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

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
