package v1

import (
	"github.com/gofiber/fiber/v2"

	"main/internal/controller/restapi/middleware"
	"main/internal/usecase"
)

func NewRouter(
	router fiber.Router,
	jwtSecret string,
	cookieDomain string,
	secureCookie bool,
	userUC usecase.UserUseCase,
	wishlistUC usecase.WishlistUseCase,
	presentUC usecase.PresentUseCase,
	uploadUC usecase.UploadUseCase,
	guestDataUC usecase.GuestDataUseCase,
	templateUC usecase.TemplateUseCase,
) {
	api := router.Group("/api/v1")

	userH := newUserHandler(userUC, uploadUC, cookieDomain, secureCookie)
	wishlistH := newWishlistHandler(wishlistUC)
	presentH := newPresentHandler(presentUC)
	uploadH := newUploadHandler(uploadUC)
	templateH := newTemplateHandler(templateUC)
	guestH := newGuestDataHandler(guestDataUC)
	systemH := newSystemTemplateHandler()
	api.Get("/system-templates", systemH.getAll)
	api.Get("/system-templates/:id", systemH.getOne)
	api.Get("/templates", middleware.JWTOptional(jwtSecret), templateH.getPublic)

	// Auth (public)
	auth := api.Group("/auth")
	auth.Post("/register", userH.register)
	auth.Post("/login", userH.login)
	auth.Post("/telegram", userH.authTelegram)

	// Auth (protected)
	authProtected := api.Group("/auth")
	authProtected.Use(middleware.JWTProtected(jwtSecret))
	authProtected.Get("/me", userH.me)
	authProtected.Post("/logout", userH.logout)

	// Wishlists (public) — статичные маршруты ПЕРЕД параметрическими.
	// Группа опознаёт гостя по куке: от этого зависит, чью бронь можно снять
	// и чей ответ показать как свой.
	guest := api.Group("")
	guest.Use(middleware.GuestIdentity(jwtSecret, cookieDomain, secureCookie))

	guest.Get("/wishlists/s/:shortId", wishlistH.getByShortID)
	guest.Get("/wishlists/:id", wishlistH.getOne)
	guest.Get("/wishlists/:wishlistId/presents", middleware.JWTOptional(jwtSecret), presentH.getAll)
	guest.Put("/presents/:id/reserve", presentH.reserve)
	guest.Put("/presents/:id/release", presentH.release)

	// Данные, которые оставляют гости. Сводка и модерация — в protected ниже,
	// по отдельным путям: один и тот же путь не может быть и публичным,
	// и защищённым.
	guest.Get("/wishlists/:wishlistId/blocks/:blockId/rsvp", guestH.myRSVP)
	guest.Post("/wishlists/:wishlistId/blocks/:blockId/rsvp", guestH.submitRSVP)
	guest.Get("/wishlists/:wishlistId/blocks/:blockId/poll", guestH.pollResults)
	guest.Post("/wishlists/:wishlistId/blocks/:blockId/poll", guestH.vote)
	guest.Get("/wishlists/:wishlistId/blocks/:blockId/playlist", guestH.tracks)
	guest.Post("/wishlists/:wishlistId/blocks/:blockId/playlist", guestH.suggestTrack)
	guest.Put("/wishlists/:wishlistId/playlist/:trackId/vote", guestH.toggleTrackVote)
	guest.Get("/wishlists/:wishlistId/blocks/:blockId/guestbook", guestH.guestbook)
	guest.Post("/wishlists/:wishlistId/blocks/:blockId/guestbook", guestH.addGuestbookEntry)
	guest.Put("/presents/:id/join", presentH.join)
	guest.Put("/presents/:id/leave", presentH.leave)

	// Protected routes
	protected := api.Group("")
	protected.Use(middleware.JWTProtected(jwtSecret))

	// User profile
	protected.Get("/users/me", userH.getProfile)
	protected.Patch("/users/me", userH.updateProfile)

	// Upload
	protected.Post("/upload", uploadH.upload)
	protected.Post("/upload/bulk", uploadH.bulkUpload)

	// Wishlists (protected)
	protected.Get("/wishlists", wishlistH.getAll)
	protected.Post("/wishlists", wishlistH.create)
	protected.Post("/wishlists/constructor", wishlistH.createConstructor)
	protected.Post("/wishlists/from-system-template", wishlistH.createFromSystemTemplate)
	protected.Put("/wishlists/:id", wishlistH.update)
	protected.Put("/wishlists/:id/blocks", wishlistH.updateBlocks)
	protected.Delete("/wishlists/:id", wishlistH.delete)

	// Ответы гостей — только владельцу вишлиста
	protected.Get("/wishlists/:wishlistId/blocks/:blockId/rsvp/summary", guestH.rsvpSummary)
	protected.Get("/wishlists/:wishlistId/blocks/:blockId/guestbook/all", guestH.ownerGuestbook)
	protected.Put("/wishlists/guestbook/:entryId/hidden", guestH.setGuestbookHidden)

	// Presents (protected)
	protected.Post("/wishlists/:wishlistId/presents", presentH.create)
	protected.Get("/presents/:id", presentH.getOne)
	protected.Put("/presents/:id", presentH.update)
	protected.Delete("/wishlists/:wishlistId/presents/:id", presentH.delete)
	protected.Put("/wishlists/:wishlistId/presents/order", presentH.reorder)
	protected.Put("/presents/:id/gifted", presentH.setGifted)

	// Templates (protected)
	protected.Get("/templates/my", templateH.getMy)
	protected.Post("/templates", templateH.create)
	protected.Patch("/templates/:id", templateH.update)
	protected.Delete("/templates/:id", templateH.delete)
	protected.Post("/wishlists/from-template/:id", templateH.createWishlistFromTemplate)
	protected.Post("/templates/:id/like", templateH.like)
	protected.Delete("/templates/:id/like", templateH.unlike)
}
