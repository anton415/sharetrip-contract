package api

import (
	"context"
	"time"

	"github.com/anton415/sharetrip-contract/gen"
	"github.com/anton415/sharetrip-contract/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (s *Server) CreateContract(ctx *fiber.Ctx) error {
	var request gen.CreateContractRequest
	if err := parseRequest(ctx, &request); err != nil ||
		request.ClientId == uuid.Nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(gen.ErrorResponse{
			Error: "invalid request body",
		})
	}

	callCtx, cancel := context.WithTimeout(
		ctx.UserContext(),
		5*time.Second,
	)
	defer cancel()

	response, err := s.contracts.CreateContract(
		callCtx,
		service.CreateContractRequest{
			ClientID:  request.ClientId,
			ExpiredAt: request.ExpiredAt,
		},
	)
	if err != nil {
		return writeError(ctx, err)
	}

	status, err := toAPIContractStatus(response.Status)
	if err != nil {
		return writeError(ctx, err)
	}

	return ctx.Status(fiber.StatusCreated).JSON(gen.CreateContractResponse{
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
