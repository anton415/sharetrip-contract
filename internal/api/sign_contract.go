package api

import (
	"github.com/anton415/sharetrip-contract/gen"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (s *Server) SignContract(ctx *fiber.Ctx) error {
	var request gen.SignContractRequest
	if err := ctx.BodyParser(&request); err != nil || request.ContractId == uuid.Nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(gen.ErrorResponse{
			Error: "invalid request body",
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(gen.SignContractResponse{
		Contract: gen.SignedContract{
			Id:     request.ContractId,
			Status: gen.Active,
		},
	})
}
