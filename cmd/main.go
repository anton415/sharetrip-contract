package main

import (
	"context"
	"errors"
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
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return fmt.Errorf("create PostgreSQL pool: %w", err)
	}
	defer pool.Close()

	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	err = pool.Ping(pingCtx)
	cancel()
	if err != nil {
		return fmt.Errorf("ping PostgreSQL: %w", err)
	}
	contracts := service.NewService(pool)

	app := fiber.New()
	app.Use(swagger.New(swagger.Config{
		BasePath: "/",
		FilePath: "./api/contract.yaml",
		Path:     "docs",
		Title:    "ShareTrip API documentation",
	}))
	contractapi.RegisterRoutes(app, contracts)

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	if err := app.Listen(addr); err != nil {
		return fmt.Errorf("listen HTTP: %w", err)
	}
	return nil
}
