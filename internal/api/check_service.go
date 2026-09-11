package api

import (
	"github.com/anton415/sharetrip-contract/gen"
	"github.com/anton415/sharetrip-contract/internal/service"
	"github.com/gofiber/fiber/v2"
)

func (s *Server) CheckService(ctx *fiber.Ctx) error {
	var request gen.CheckServiceRequest
	if err := parseRequest(ctx, &request); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(gen.ErrorResponse{Error: "invalid request body"})
	}
	response, err := s.contractService.CheckService(ctx.UserContext(), service.CheckServiceRequest{
		ClientID:    request.ClientId,
		ServiceCode: request.ServiceCode,
	})
	if err != nil {
		return writeError(ctx, err)
	}
	return ctx.Status(fiber.StatusOK).JSON(gen.CheckServiceResponse{
		Allowed: response.Allowed,
		Reason:  gen.CheckServiceResponseReason(response.Reason),
	})
}
