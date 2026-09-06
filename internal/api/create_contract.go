package api

import (
	"time"

	"github.com/anton415/sharetrip-contract/gen"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (s *Server) CreateContract(ctx *fiber.Ctx) error {
	var request gen.CreateContractRequest
	if err := ctx.BodyParser(&request); err != nil || request.ClientId == uuid.Nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(gen.ErrorResponse{
			Error: "invalid request body",
		})
	}

	now := time.Now().UTC()
	contract := gen.Contract{
		Id:        uuid.New(),
		ClientId:  request.ClientId,
		Status:    gen.Draft,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiredAt: request.ExpiredAt,
	}

	return ctx.Status(fiber.StatusCreated).JSON(gen.CreateContractResponse{
		Contract: contract,
	})
}
