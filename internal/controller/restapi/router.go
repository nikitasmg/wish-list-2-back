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
	guestDataUC usecase.GuestDataUseCase,
	templateUC usecase.TemplateUseCase,
	santaUC usecase.SantaUseCase,
) {
	app.Use(logger.New())
	app.Use(compress.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.App.CORSOrigin,
		// If-Match несёт версию вишлиста при сохранении блоков и метаданных.
		// Без него браузер отрезает заголовок на preflight, и проверка версии
		// перестаёт работать именно там, где она нужна: фронт живёт на другом
		// поддомене, а падает это молча — запрос проходит, конфликт не ловится.
		// X-Santa-Token — секрет участника Тайного Санты без аккаунта.
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Custom-Header, If-Match, X-Santa-Token",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS, PATCH",
		AllowCredentials: true,
		ExposeHeaders:    "Content-Length, X-Knowledge-Base",
		MaxAge:           3600,
	}))
	app.Use(recover.New())
	app.Use(limiter.New(limiter.Config{
		Max:        10,
		Expiration: 1 * time.Second,
		// Вебхук бота защищён секретом; апдейты Telegram идут с немногих IP.
		Next: func(c *fiber.Ctx) bool { return c.Path() == "/api/v1/telegram/webhook" },
	}))
	app.Use(middleware.CookieToHeader())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// До основного роутера: см. комментарий у NewSantaRouter.
	v1.NewSantaRouter(app, cfg.Auth.JWTSecret, cfg.Notify.TelegramWebhookSecret, santaUC)
	v1.NewRouter(app, cfg.Auth.JWTSecret, cfg.Auth.CookieDomain, cfg.App.Env == "production", userUC, wishlistUC, presentUC, uploadUC, guestDataUC, templateUC)
}
