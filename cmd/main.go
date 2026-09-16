package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	contractapi "github.com/anton415/sharetrip-contract/internal/api"
	"github.com/anton415/sharetrip-contract/internal/service"

	"github.com/gofiber/contrib/swagger"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatal(fmt.Errorf("create PostgreSQL pool: %w", err))
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	err = pool.Ping(pingCtx)
	cancel()
	if err != nil {
		pool.Close()
		log.Fatal(fmt.Errorf("ping PostgreSQL: %w", err))
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

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	if err := app.Listen(addr); err != nil {
		pool.Close()
		log.Fatal(fmt.Errorf("listen HTTP: %w", err))
	}
}
