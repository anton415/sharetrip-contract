package api

import (
	"fmt"

	"github.com/anton415/sharetrip-contract/gen"
	"github.com/anton415/sharetrip-contract/internal/service"
	"github.com/gofiber/fiber/v2"
)

type checkServiceResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason"`
}

func (s *Server) CheckService(ctx *fiber.Ctx) error {
	request := ctx.Locals(checkServiceRequestKey{}).(gen.CheckServiceJSONBody)
	response, err := s.contractService.CheckService(ctx.UserContext(), service.CheckServiceRequest{
		ClientID:    request.ClientId,
		ServiceCode: request.ServiceCode,
	})
	if err != nil {
		return writeError(ctx, fmt.Errorf("check service: %w", err))
	}
	return ctx.Status(fiber.StatusOK).JSON(checkServiceResponse{
		Allowed: response.Allowed,
		Reason:  response.Reason,
	})
}
