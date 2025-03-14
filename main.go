package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"log"
	"main/common/config"
	"main/database"
	"main/pkg/minio"
	"main/router"
	"os"
	"time"
)

func main() {
	config.LoadConfig()
	// инициализируем базу данных
	if err := database.Connect(); err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	// Инициализация соединения с Minio
	minioClient := minio.NewMinioClient()
	err := minioClient.InitMinio()

	if err != nil {
		log.Fatalf("Ошибка инициализации Minio: %v", err)
	}
	app := fiber.New()
	app.Use(logger.New())
	app.Use(compress.New())
	// Получаем значение переменной окружения PROD_URL
	prodURL := os.Getenv("PROD_URL")
	if prodURL == "" {
		prodURL = "http://localhost:3000" // Значение по умолчанию для разработки
	}
	allowOrigins := prodURL + ",http://localhost:3000"

	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowOrigins, // или "*" для всех
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Custom-Header",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS, PATCH",
		AllowCredentials: true,
		ExposeHeaders:    "Content-Length, X-Knowledge-Base",
		MaxAge:           3600, // 1 час
	}))

	app.Use(recover.New())
	app.Use(limiter.New(limiter.Config{
		Max:        10,
		Expiration: 1 * time.Second,
	}))

	app.Use(func(c *fiber.Ctx) error {
		// Извлекаем токен из куки
		token := c.Cookies("token")

		// Если токен не найден, продолжаем дальше
		if token == "" {
			return c.Next()
		}

		// Устанавливаем токен в заголовок для последующего middleware
		c.Request().Header.Set("Authorization", "Bearer "+token)
		return c.Next()
	})

	// Get the PORT from heroku env
	port := os.Getenv("PORT")

	// Verify if heroku provided the port or not
	if os.Getenv("PORT") == "" {
		port = "8080"
	}

	router.SetupRoutes(app, minioClient)
	log.Fatal(app.Listen(":" + port))
}
