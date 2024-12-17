package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"log"
	"main/router"
	"os"
)

func main() {
	app := fiber.New()
	app.Use(cors.New())

	// Get the PORT from heroku env
	port := os.Getenv("PORT")

	// Verify if heroku provided the port or not
	if os.Getenv("PORT") == "" {
		port = "3000"
	}

	router.SetupRoutes(app)
	log.Fatal(app.Listen(":" + port))
}
