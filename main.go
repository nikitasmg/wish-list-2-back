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
	app := fiber.New(fiber.Config{
		//Prefork: true,
	})
	app.Use(logger.New())
	app.Use(compress.New())
	app.Use(cors.New())
	app.Use(recover.New())
	app.Use(limiter.New(limiter.Config{
		Max:        1,
		Expiration: 1 * time.Second,
	}))

	// Get the PORT from heroku env
	port := os.Getenv("PORT")

	// Verify if heroku provided the port or not
	if os.Getenv("PORT") == "" {
		port = "8080"
	}

	router.SetupRoutes(app, minioClient)
	log.Fatal(app.Listen(":" + port))
}
