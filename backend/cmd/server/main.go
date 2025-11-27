package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/dpolishuk/neograph/backend/internal/config"
)

func main() {
	cfg := config.Load()

	app := fiber.New(fiber.Config{
		AppName: "NeoGraph API",
	})

	// Health check
	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"service": "neograph-backend",
		})
	})

	// API routes will be added here
	api := app.Group("/api")
	api.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "NeoGraph API v1",
		})
	})

	log.Printf("Starting NeoGraph backend on port %s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}
