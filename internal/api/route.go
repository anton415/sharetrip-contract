package api

import (
	"context"

	"github.com/anton415/sharetrip-contract/gen"
	"github.com/anton415/sharetrip-contract/internal/service"
	"github.com/gofiber/fiber/v2"
)

type ContractService interface {
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
	gen.RegisterHandlers(router, NewServer(contractService))
}
