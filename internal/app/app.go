package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"

	"main/config"
	"main/internal/controller/restapi"
	"main/internal/repo/persistent"
	"main/internal/usecase"
	guestDataUC "main/internal/usecase/guestdata"
	presentUC "main/internal/usecase/present"
	santaUC "main/internal/usecase/santa"
	templateUC "main/internal/usecase/template"
	uploadUC "main/internal/usecase/upload"
	userUC "main/internal/usecase/user"
	wishlistUC "main/internal/usecase/wishlist"
	"main/pkg/hasher"
	"main/pkg/mailer"
	minioPkg "main/pkg/minio"
	"main/pkg/postgres"
	"main/pkg/telegram"
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
		&persistent.SantaEmailCodeModel{}, &persistent.SantaTgLinkModel{}, &persistent.SantaNotificationModel{},
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
	if err := persistent.BackfillSantaPendingEmail(db); err != nil {
		log.Fatalf("backfill santa pending_email: %v", err)
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
	// Уведомления Санты: без SMTP и токена бота — в лог (разработка).
	production := cfg.App.Env == "production"
	var mail usecase.Mailer = mailer.NewLog()
	if cfg.Notify.SMTPHost != "" {
		smtpMailer, err := mailer.NewSMTP(mailer.Config{
			Host: cfg.Notify.SMTPHost, Port: cfg.Notify.SMTPPort,
			User: cfg.Notify.SMTPUser, Password: cfg.Notify.SMTPPassword, From: cfg.Notify.MailFrom,
		})
		if err != nil {
			log.Fatalf("mailer: %v", err)
		}
		mail = smtpMailer
	} else if production {
		log.Println("WARNING: SMTP_HOST не задан — коды на почту Санты не отправляются (ответ 503), письма уведомлений уходят в повтор и failed")
	} else {
		log.Println("WARNING: SMTP_HOST не задан — письма Санты уходят в лог")
	}
	var bot usecase.TelegramSender = telegram.NewLog()
	if cfg.Notify.TelegramBotToken != "" {
		bot = telegram.New(cfg.Notify.TelegramBotToken)
	} else {
		log.Println("WARNING: SANTA_BOT_TOKEN/BOT_TOKEN не задан — сообщения Санты уходят в лог")
	}
	if cfg.Notify.TelegramBotUsername == "" {
		log.Println("WARNING: BOT_USERNAME не задан — ссылки на бота Санты не выдаются (ответ 503)")
	}
	if cfg.Notify.TelegramWebhookSecret == "" {
		log.Println("WARNING: TELEGRAM_WEBHOOK_SECRET не задан — вебхук бота выключен, ссылки на бота Санты не выдаются (ответ 503)")
	}
	// Процесс не роняем: переменные могут задать позже, а остальной сервис
	// должен работать. Ненастроенный канал отвечает 503, а не «код отправлен»
	// или ссылкой на бота, который никому не ответит.
	var santaOpts []santaUC.Option
	if cfg.Notify.SMTPHost != "" || !production {
		santaOpts = append(santaOpts, santaUC.WithMailer(mail))
	}
	linkBot := cfg.Notify.TelegramBotUsername
	if cfg.Notify.TelegramWebhookSecret == "" || cfg.Notify.TelegramBotToken == "" {
		linkBot = "" // /start по ссылке до нас не дойдёт
	}
	santaOpts = append(santaOpts, santaUC.WithTelegram(bot, linkBot))
	santaUseCase := santaUC.New(santaRepo, userRepo, santaOpts...)
	notifyMail, notifyBot := notifierChannels(production, cfg.Notify.SMTPHost != "", cfg.Notify.TelegramBotToken != "", mail, bot)
	notifyCtx, stopNotify := context.WithCancel(context.Background())
	defer stopNotify()
	notifierDone := make(chan struct{})
	go func() {
		defer close(notifierDone)
		santaUC.NewNotifier(santaRepo, notifyMail, notifyBot, cfg.Notify.SantaPublicURL).Run(notifyCtx, 5*time.Second)
	}()

	// HTTP server
	app := fiber.New(fiber.Config{
		BodyLimit: 15 * 1024 * 1024, // 15MB — headroom for multipart overhead
		// За Traefik адрес соединения — адрес прокси; реальный IP клиента он
		// кладёт в X-Real-Ip. Заголовку верим только от прокси из частных
		// сетей (dokploy-network), иначе лимитеры стали бы общими на весь сайт.
		ProxyHeader:             "X-Real-Ip",
		EnableTrustedProxyCheck: true,
		TrustedProxies:          []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
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
	stopNotify()
	<-notifierDone // дать обработчику дописать статус текущей отправки
	if err := app.Shutdown(); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}

// notifierChannels — каналы для обработчика очереди. В продакшене без SMTP или
// токена бота канала нет (nil), а не лог-заглушка: заглушка молча отмечала бы
// уведомления отправленными, а так они уходят в повтор и потом в failed.
func notifierChannels(production, smtpSet, botSet bool, mail usecase.Mailer, bot usecase.TelegramSender) (usecase.Mailer, usecase.TelegramSender) {
	if production && !smtpSet {
		mail = nil
	}
	if production && !botSet {
		bot = nil
	}
	return mail, bot
}
