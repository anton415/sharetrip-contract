package api

import (
	"github.com/anton415/sharetrip-contract/gen"
	"github.com/anton415/sharetrip-contract/internal/service"
	"github.com/gofiber/fiber/v2"
)

func (s *Server) CheckService(ctx *fiber.Ctx) error {
	var request gen.CheckServiceJSONBody
	if err := parseRequest(ctx, &request); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(gen.ErrorResponse{Error: "invalid request body"})
	}
	logger := s.logger.With(
		"service", "contract",
		"operation", "CheckPermission",
		"request_id", ctx.Get("X-Request-ID"),
		"correlation_id", ctx.Get("X-Correlation-ID"),
		"trip_id", ctx.Get("X-Trip-ID"),
		"user_id", request.ClientId.String(),
	)
	response, err := s.contractService.CheckService(ctx.UserContext(), service.CheckServiceRequest{
		ClientID:    request.ClientId,
		ServiceCode: request.ServiceCode,
	})
	if err != nil {
		logger.ErrorContext(ctx.UserContext(), "permission check", "result", "error")
		return writeError(ctx, err)
	}
	result := "denied"
	if response.Allowed {
		result = "allowed"
	}
	logger.InfoContext(ctx.UserContext(), "permission check", "result", result)
	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"allowed": response.Allowed,
		"reason":  response.Reason,
	})
}
