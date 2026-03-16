package app

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"

	"main/config"
	"main/internal/controller/restapi"
	"main/internal/repo/persistent"
	parseUC "main/internal/usecase/parse"
	presentUC "main/internal/usecase/present"
	uploadUC "main/internal/usecase/upload"
	userUC "main/internal/usecase/user"
	wishlistUC "main/internal/usecase/wishlist"
	"main/pkg/hasher"
	minioPkg "main/pkg/minio"
	"main/pkg/postgres"
)

func Run(cfg *config.Config) {
	// Database
	db, err := postgres.New(cfg.DB)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}

	// AutoMigrate
	if err := db.AutoMigrate(
		&persistent.UserModel{},
		&persistent.WishlistModel{},
		&persistent.PresentModel{},
		&persistent.ParseRateLimitModel{},
		&persistent.PresentMetaModel{},
	); err != nil {
		log.Fatalf("automigrate: %v", err)
	}

	// MinIO
	fileStorage, err := minioPkg.New(cfg.Minio, cfg.App.CORSOrigin)
	if err != nil {
		log.Fatalf("minio: %v", err)
	}

	// Repositories
	userRepo := persistent.NewUserRepo(db)
	wishlistRepo := persistent.NewWishlistRepo(db)
	presentRepo := persistent.NewPresentRepo(db)
	presentMetaRepo := persistent.NewPresentMetaRepo(db)
	rateLimitRepo := persistent.NewParseRateLimitRepo(db)

	// Hasher
	pwHasher := hasher.New()

	// Use Cases
	userUseCase := userUC.New(userRepo, pwHasher, cfg.Auth.JWTSecret, cfg.Auth.BotToken)
	wishlistUseCase := wishlistUC.New(wishlistRepo, fileStorage)
	presentUseCase := presentUC.New(presentRepo, wishlistRepo, fileStorage, presentMetaRepo)
	uploadUseCase := uploadUC.New(fileStorage)
	httpClient := &http.Client{Timeout: 15 * time.Second}
	parseUseCase := parseUC.NewParseUseCase(rateLimitRepo, httpClient)

	// HTTP server
	app := fiber.New(fiber.Config{
		BodyLimit: 15 * 1024 * 1024, // 15MB — headroom for multipart overhead
	})
	restapi.NewRouter(app, cfg, userUseCase, wishlistUseCase, presentUseCase, uploadUseCase, parseUseCase)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := app.Listen(fmt.Sprintf(":%s", cfg.App.Port)); err != nil {
			log.Printf("server stopped: %v", err)
		}
	}()

	log.Printf("Server started on :%s", cfg.App.Port)
	<-quit
	log.Println("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}
