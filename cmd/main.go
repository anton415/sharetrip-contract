package main

import (
	"context"
	"fmt"
	"log"
	"time"

	contractapi "github.com/anton415/sharetrip-contract/internal/api"
	"github.com/anton415/sharetrip-contract/internal/config"
	"github.com/anton415/sharetrip-contract/internal/service"

	"github.com/gofiber/contrib/swagger"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal("create PostgreSQL pool failed")
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	err = pool.Ping(pingCtx)
	cancel()
	if err != nil {
		pool.Close()
		log.Fatal("ping PostgreSQL failed")
	}
	contractService := service.NewService(pool)

	app := fiber.New()
	app.Use(swagger.New(swagger.Config{
		BasePath: "/",
		FilePath: "./api/contract.bundle.yaml",
		Path:     "docs",
		Title:    "ShareTrip API documentation",
	}))
	contractapi.RegisterRoutes(app, contractService)

	if err := app.Listen(cfg.HTTPAddr); err != nil {
		pool.Close()
		log.Fatal(fmt.Errorf("listen HTTP: %w", err))
	}
}
