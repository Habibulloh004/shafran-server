package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/example/shafran/internal/config"
	"github.com/example/shafran/internal/database"
	"github.com/example/shafran/internal/routes"
	"github.com/example/shafran/internal/services"
)

func main() {
	cfg := config.Load()
	db := database.Connect(cfg.DatabaseURL)

	app := fiber.New(fiber.Config{
		AppName: "Shafran Backend",
	})

	app.Use(recover.New())
	app.Use(logger.New())

	routes.Register(app, db, cfg)

	if _, err := services.GetBillzToken(); err != nil {
		log.Printf("Billz token warm-up failed: %v", err)
	}

	log.Printf("Starting server on :%s", cfg.AppPort)
	if err := app.Listen(":" + cfg.AppPort); err != nil {
		log.Fatalf("fiber.Listen error: %v", err)
	}
}
