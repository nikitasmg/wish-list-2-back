package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	jwtware "github.com/gofiber/jwt/v2"
	"main/handler"
	"main/pkg/minio"
	"os"
)

// SetupRoutes setup router api
func SetupRoutes(app *fiber.App, minio minio.Client) {

	presentHandlers := handler.NewPresentService(minio)
	wishlistHandlers := handler.NewWishlistHandlers(minio)

	// Группа обработчиков, которые доступны неавторизованным пользователям
	publicGroup := app.Group("/api", logger.New())
	publicGroup.Post("/register", handler.Register)
	publicGroup.Post("/login", handler.Login)

	// Публичные ручки для отображения вишлиста
	publicGroup.Get("/present/:wishlistId", presentHandlers.GetAll)
	publicGroup.Put("/present/reserve/:id", presentHandlers.Reserve)
	publicGroup.Put("/present/release/:id", presentHandlers.Release)

	jwtSecret := os.Getenv("JWT_SECRET")

	// Группа обработчиков, которые требуют авторизации
	authorizedGroup := app.Group("/api", logger.New())
	authorizedGroup.Use(jwtware.New(jwtware.Config{
		SigningKey: []byte(jwtSecret),
		ContextKey: "user",
	}))

	// User
	authorizedGroup.Get("/me", handler.Me)

	// Wishlist
	authorizedGroup.Get("/wishlist", wishlistHandlers.GetAll)
	authorizedGroup.Get("/wishlist/:id", wishlistHandlers.GetOne)
	authorizedGroup.Post("/wishlist", wishlistHandlers.Create)
	authorizedGroup.Delete("/wishlist/:id", wishlistHandlers.Delete)
	authorizedGroup.Patch("/wishlist/:id", wishlistHandlers.Update)

	// Presents
	authorizedGroup.Post("/present/:wishlistId", presentHandlers.Create)
	authorizedGroup.Delete("/present/:id", presentHandlers.Delete)
	authorizedGroup.Patch("/present/:id", presentHandlers.Update)
}
