package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
	"log"
	"main/database"
	"main/router"
	"os"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	// инициализируем базу данных
	if err := database.Connect(); err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	app := fiber.New(fiber.Config{
		Prefork: true,
	})
	app.Use(logger.New())
	app.Use(compress.New())
	app.Use(cors.New())
	app.Use(recover.New())
	app.Use(limiter.New())

	// Get the PORT from heroku env
	port := os.Getenv("PORT")

	// Verify if heroku provided the port or not
	if os.Getenv("PORT") == "" {
		port = "3000"
	}

	router.SetupRoutes(app)
	log.Fatal(app.Listen(":" + port))
}
