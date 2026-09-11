package api

import (
	"context"

	"github.com/anton415/sharetrip-contract/gen"
	"github.com/anton415/sharetrip-contract/internal/service"
	"github.com/gofiber/fiber/v2"
)

type ContractService interface {
	CheckService(context.Context, service.CheckServiceRequest) (service.CheckServiceResponse, error)

	UpsertContractServices(context.Context, service.UpsertContractServicesRequest) error

	GetContract(
		context.Context,
		service.GetContractRequest,
	) (service.GetContractResponse, error)

	GetActiveContract(
		context.Context,
		service.GetActiveContractRequest,
	) (service.GetContractResponse, error)

	CreateContract(
		context.Context,
		service.CreateContractRequest,
	) (service.CreateContractResponse, error)

	SignContract(
		context.Context,
		service.SignContractRequest,
	) (service.SignContractResponse, error)
}

type Server struct {
	contractService ContractService
}

var _ gen.ServerInterface = (*Server)(nil)

func NewServer(contractService ContractService) *Server {
	return &Server{contractService: contractService}
}

func RegisterRoutes(router fiber.Router, contractService ContractService) {
	router.Get("/health", func(ctx *fiber.Ctx) error {
		return ctx.SendStatus(fiber.StatusOK)
	})

	gen.RegisterHandlers(router, NewServer(contractService))
}
