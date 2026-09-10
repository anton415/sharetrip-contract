package api

import (
	"github.com/anton415/sharetrip-contract/gen"
	"github.com/anton415/sharetrip-contract/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (s *Server) SignContract(ctx *fiber.Ctx) error {
	var request gen.SignContractRequest
	if err := parseRequest(ctx, &request); err != nil || request.ContractId == uuid.Nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(gen.ErrorResponse{
			Error: "invalid request body",
		})
	}

	response, err := s.contractService.SignContract(
		ctx.UserContext(),
		service.SignContractRequest{
			ContractID: request.ContractId,
		},
	)
	if err != nil {
		return writeError(ctx, err)
	}

	return ctx.Status(fiber.StatusOK).JSON(gen.SignContractResponse{
		Contract: gen.SignedContract{
			Id:     response.ID,
			Status: gen.ContractStatus(response.Status),
		},
	})
}
