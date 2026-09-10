package api

import (
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

	response, err := s.contractService.CreateContract(
		ctx.UserContext(),
		service.CreateContractRequest{
			ClientID:  request.ClientId,
			ExpiredAt: request.ExpiredAt,
		},
	)
	if err != nil {
		return writeError(ctx, err)
	}

	return ctx.Status(fiber.StatusCreated).JSON(gen.CreateContractResponse{
		Contract: gen.Contract{
			Id:        response.ID,
			ClientId:  response.ClientID,
			Status:    gen.ContractStatus(response.Status),
			CreatedAt: response.CreatedAt,
			UpdatedAt: response.UpdatedAt,
			ExpiredAt: response.ExpiredAt,
		},
	})
}
