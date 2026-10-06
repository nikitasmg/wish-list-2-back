package app

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"

	"main/config"
	"main/internal/controller/restapi"
	"main/internal/repo/persistent"
	guestDataUC "main/internal/usecase/guestdata"
	presentUC "main/internal/usecase/present"
	santaUC "main/internal/usecase/santa"
	templateUC "main/internal/usecase/template"
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
		&persistent.WishlistViewModel{},
		&persistent.RSVPResponseModel{},
		&persistent.PollVoteModel{},
		&persistent.PollChoiceModel{},
		&persistent.PollGuestOptionModel{},
		&persistent.PlaylistTrackModel{},
		&persistent.PlaylistVoteModel{},
		&persistent.GuestbookEntryModel{},
		&persistent.PresentMetaModel{},
		&persistent.TemplateModel{},
		&persistent.TemplateLikeModel{},
		&persistent.SantaRoomModel{}, &persistent.SantaParticipantModel{}, &persistent.SantaAssignmentModel{},
	); err != nil {
		log.Fatalf("automigrate: %v", err)
	}

	// AutoMigrate добавляет колонки, но не переносит данные: дату праздника
	// нужно достать из location вручную.
	if err := persistent.BackfillEventDate(db); err != nil {
		log.Fatalf("backfill event_date: %v", err)
	}
	if err := persistent.BackfillPresentLinks(db); err != nil {
		log.Fatalf("backfill present links: %v", err)
	}
	if err := persistent.BackfillPollChoices(db); err != nil {
		log.Fatalf("backfill poll choices: %v", err)
	}

	// MinIO
	fileStorage, err := minioPkg.New(cfg.Minio, cfg.App.MinioPublicURL)
	if err != nil {
		log.Fatalf("minio: %v", err)
	}
	fileStorage = minioPkg.NewOptimizing(fileStorage)

	// Repositories
	userRepo := persistent.NewUserRepo(db)
	wishlistRepo := persistent.NewWishlistRepo(db)
	presentRepo := persistent.NewPresentRepo(db)
	guestDataRepo := persistent.NewGuestDataRepo(db)
	presentMetaRepo := persistent.NewPresentMetaRepo(db)
	templateRepo := persistent.NewTemplateRepo(db)
	santaRepo := persistent.NewSantaRepo(db)

	// Hasher
	pwHasher := hasher.New()

	// Use Cases
	userUseCase := userUC.New(userRepo, pwHasher, cfg.Auth.JWTSecret, cfg.Auth.BotToken)
	wishlistUseCase := wishlistUC.New(wishlistRepo, fileStorage)
	presentUseCase := presentUC.New(presentRepo, wishlistRepo, fileStorage, presentMetaRepo)
	uploadUseCase := uploadUC.New(fileStorage)
	guestDataUseCase := guestDataUC.New(guestDataRepo, wishlistRepo)
	templateUseCase := templateUC.New(templateRepo, wishlistRepo)
	santaUseCase := santaUC.New(santaRepo, userRepo)

	// HTTP server
	app := fiber.New(fiber.Config{
		BodyLimit: 15 * 1024 * 1024, // 15MB — headroom for multipart overhead
	})
	restapi.NewRouter(app, cfg, userUseCase, wishlistUseCase, presentUseCase, uploadUseCase, guestDataUseCase, templateUseCase, santaUseCase)

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
