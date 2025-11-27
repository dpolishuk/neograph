package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/dpolishuk/neograph/backend/internal/api"
	"github.com/dpolishuk/neograph/backend/internal/config"
	"github.com/dpolishuk/neograph/backend/internal/db"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
)

func main() {
	cfg := config.Load()

	// Connect to Neo4j
	dbClient, err := db.NewNeo4jClient(context.Background(), db.Neo4jConfig{
		URI:      cfg.Neo4jURI,
		Username: cfg.Neo4jUser,
		Password: cfg.Neo4jPass,
	})
	if err != nil {
		log.Fatalf("Failed to connect to Neo4j: %v", err)
	}
	defer dbClient.Close()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "NeoGraph API",
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	}))

	// Health check
	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "neograph-backend",
		})
	})

	// Setup API routes
	handler := api.NewHandler(cfg, dbClient)
	defer handler.Close()
	api.SetupRoutes(app, handler)

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down...")
		app.Shutdown()
	}()

	log.Printf("Starting NeoGraph backend on port %s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
