package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	_ "github.com/joho/godotenv/autoload"

	"github.com/dominikwinter/slackgpt/internal/router"
)

func main() {
	var level slog.Leveler

	if os.Getenv("DEBUG") == "true" {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	app := fiber.New(fiber.Config{
		ReadTimeout:  30 * time.Second,
		IdleTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	})

	app.Use(recover.New(recover.Config{EnableStackTrace: true}))
	app.Use(logger.New())

	router.Setup(app, log)

	log.Error("error: %w", app.Listen(":3000"))
}
