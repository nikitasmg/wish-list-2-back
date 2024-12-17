package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"main/handler"
)

// SetupRoutes setup router api
func SetupRoutes(app *fiber.App) {
	// Middleware
	api := app.Group("/api", logger.New())

	// Cards
	product := api.Group("/card")
	product.Get("/", handler.GetAllCards)
	product.Get("/:id", handler.GetCard)
	product.Post("/", handler.CreateCard)
	product.Delete("/:id", handler.DeleteCard)
	product.Put("/:id", handler.UpdateCard)
}
