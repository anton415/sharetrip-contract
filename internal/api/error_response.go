package api

import (
	"errors"
	"log/slog"

	"github.com/anton415/sharetrip-contract/gen"
	"github.com/anton415/sharetrip-contract/internal/domain"
	"github.com/anton415/sharetrip-contract/internal/repository"
	"github.com/gofiber/fiber/v2"
)

func writeError(ctx *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	message := "internal server error"

	switch {
	case errors.Is(err, domain.ErrInvalidClientID):
		status = fiber.StatusBadRequest
		message = "invalid client id"

	case errors.Is(err, domain.ErrInvalidContractID):
		status = fiber.StatusBadRequest
		message = "invalid contract id"

	case errors.Is(err, repository.ErrContractNotFound):
		status = fiber.StatusNotFound
		message = "contract not found"

	case errors.Is(err, domain.ErrInvalidContractTransition):
		status = fiber.StatusConflict
		message = "contract cannot be signed in its current state"

	default:
		slog.Error("contract request failed", "error", err)
	}

	return ctx.Status(status).JSON(gen.ErrorResponse{
		Error: message,
	})
}
