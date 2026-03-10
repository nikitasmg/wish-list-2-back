package restapi

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"main/config"
	"main/internal/controller/restapi/middleware"
	v1 "main/internal/controller/restapi/v1"
	"main/internal/usecase"
)

func NewRouter(
	app *fiber.App,
	cfg *config.Config,
	userUC usecase.UserUseCase,
	wishlistUC usecase.WishlistUseCase,
	presentUC usecase.PresentUseCase,
	uploadUC usecase.UploadUseCase,
) {
	app.Use(logger.New())
	app.Use(compress.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.App.CORSOrigin,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Custom-Header",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS, PATCH",
		AllowCredentials: true,
		ExposeHeaders:    "Content-Length, X-Knowledge-Base",
		MaxAge:           3600,
	}))
	app.Use(recover.New())
	app.Use(limiter.New(limiter.Config{
		Max:        10,
		Expiration: 1 * time.Second,
	}))
	app.Use(middleware.CookieToHeader())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	v1.NewRouter(app, cfg.Auth.JWTSecret, cfg.Auth.CookieDomain, userUC, wishlistUC, presentUC, uploadUC)
}
