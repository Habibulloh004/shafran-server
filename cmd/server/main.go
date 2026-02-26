package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
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
		// 200 MB max request body size
		BodyLimit: 200 * 1024 * 1024,
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Content-Type,Authorization",
	}))

	// Ensure uploads directory exists
	os.MkdirAll("uploads/banners", 0755)

	// Serve uploaded files
	app.Static("/uploads", "./uploads")

	routes.Register(app, db, cfg)

	if _, err := services.GetBillzToken(); err != nil {
		log.Printf("Billz token warm-up failed: %v", err)
	}

	log.Printf("Starting server on :%s", cfg.AppPort)
	if err := app.Listen(":" + cfg.AppPort); err != nil {
		log.Fatalf("fiber.Listen error: %v", err)
	}
}
