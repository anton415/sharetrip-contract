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
	contracts ContractService
}

var _ gen.ServerInterface = (*Server)(nil)

func NewServer(contracts ContractService) *Server {
	return &Server{contracts: contracts}
}

func RegisterRoutes(router fiber.Router, contracts ContractService) {
	gen.RegisterHandlers(router, NewServer(contracts))
}
