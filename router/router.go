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
	publicGroup := app.Group("", logger.New())
	publicGroup.Post("/register", handler.Register)
	publicGroup.Post("/login", handler.Login)
	publicGroup.Post("/auth", handler.Authenticate)

	// Публичные ручки для отображения вишлиста
	publicGroup.Get("/wishlist/:wishlistId/presents", presentHandlers.GetAll)
	publicGroup.Put("/present/reserve/:id", presentHandlers.Reserve)
	publicGroup.Put("/present/release/:id", presentHandlers.Release)
	publicGroup.Get("/wishlist/:id", wishlistHandlers.GetOne)

	jwtSecret := os.Getenv("JWT_SECRET")

	// Группа обработчиков, которые требуют авторизации
	authorizedGroup := app.Group("", logger.New())
	authorizedGroup.Use(jwtware.New(jwtware.Config{
		SigningKey: []byte(jwtSecret),
		ContextKey: "user",
	}))

	// User
	authorizedGroup.Get("/me", handler.Me)
	authorizedGroup.Get("/logout", handler.Logout)

	// Wishlist
	authorizedGroup.Get("/wishlist", wishlistHandlers.GetAll)
	authorizedGroup.Post("/wishlist", wishlistHandlers.Create)
	authorizedGroup.Delete("/wishlist/:id", wishlistHandlers.Delete)
	authorizedGroup.Put("/wishlist/:id", wishlistHandlers.Update)

	// Presents
	authorizedGroup.Post("/present/:wishlistId", presentHandlers.Create)
	authorizedGroup.Get("/present/:id", presentHandlers.GetOne)
	authorizedGroup.Delete("/present/:wishlistId/:id", presentHandlers.Delete)
	authorizedGroup.Put("/present/:id", presentHandlers.Update)
}
