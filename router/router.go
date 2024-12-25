package router

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	jwtware "github.com/gofiber/jwt/v2"
	"main/handler"
	"os"
)

// SetupRoutes setup router api
func SetupRoutes(app *fiber.App) {
	// Группа обработчиков, которые доступны неавторизованным пользователям
	publicGroup := app.Group("/api", logger.New())
	publicGroup.Post("/register", handler.Register)
	publicGroup.Post("/login", handler.Login)

	jwtSecret := os.Getenv("JWT_SECRET")

	// Группа обработчиков, которые требуют авторизации
	authorizedGroup := app.Group("/api", logger.New())
	authorizedGroup.Use(jwtware.New(jwtware.Config{
		SigningKey: []byte(jwtSecret),
		ContextKey: "user",
		SuccessHandler: func(c *fiber.Ctx) error {
			// Обработка успешного валидации JWT
			return c.Next()
		},
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// Логируем ошибку
			fmt.Println(c.GetReqHeaders())
			fmt.Println("JWT Error:", err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired JWT"})
		},
	}))

	// User
	authorizedGroup.Get("/me", handler.Me)

	// WishList
	authorizedGroup.Get("/wishlist", handler.GetAllWishlists)
	authorizedGroup.Post("/wishlist", handler.CreateWishlist)
	authorizedGroup.Get("/wishlist/:id", handler.GetWishList)

	// Presents
	authorizedGroup.Post("/present/:id", handler.CreatePresent)
	authorizedGroup.Get("/present/:id", handler.GetPresents)
}
