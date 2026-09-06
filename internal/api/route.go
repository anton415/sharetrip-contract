package api

import (
	"github.com/anton415/sharetrip-contract/gen"

	"github.com/gofiber/fiber/v2"
)

type Server struct{}

var _ gen.ServerInterface = (*Server)(nil)

func NewServer() *Server {
	return &Server{}
}

func RegisterRoutes(router fiber.Router) {
	gen.RegisterHandlers(router, NewServer())
}
