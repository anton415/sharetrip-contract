package main

import (
	"log"

	contractapi "github.com/anton415/sharetrip-contract/internal/api"

	"github.com/gofiber/contrib/swagger"
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()
	app.Use(swagger.New(swagger.Config{
		BasePath: "/",
		FilePath: "./api/contract.yaml",
		Path:     "docs",
		Title:    "ShareTrip API documentation",
	}))
	contractapi.RegisterRoutes(app)

	log.Fatal(app.Listen(":8080"))
}
