package api

import (
	"context"
	"log/slog"

	"github.com/anton415/sharetrip-contract/gen"
	"github.com/gofiber/fiber/v2"
)

type requestLoggerKey struct{}
type checkServiceRequestKey struct{}

func (s *Server) checkServiceMiddleware(ctx *fiber.Ctx, next fiber.Handler) error {
	if ctx.Route().Path != "/contracts/check-service" {
		return next(ctx)
	}

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
		"client_id", request.ClientId.String(),
	)
	ctx.SetUserContext(context.WithValue(ctx.UserContext(), requestLoggerKey{}, logger))
	ctx.Locals(checkServiceRequestKey{}, request)
	return next(ctx)
}

func requestLogger(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(requestLoggerKey{}).(*slog.Logger)
	if !ok || logger == nil {
		return slog.Default()
	}
	return logger
}
