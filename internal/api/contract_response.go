package api

import (
	"github.com/anton415/sharetrip-contract/gen"
	"github.com/anton415/sharetrip-contract/internal/service"
	"github.com/gofiber/fiber/v2"
)

func writeContract(ctx *fiber.Ctx, response service.GetContractResponse) error {
	return ctx.Status(fiber.StatusOK).JSON(gen.GetContractResponse{
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
