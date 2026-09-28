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
) {
	api := router.Group("/api/v1")

	userH := newUserHandler(userUC, cookieDomain, secureCookie)
	wishlistH := newWishlistHandler(wishlistUC)
	presentH := newPresentHandler(presentUC)
	uploadH := newUploadHandler(uploadUC)
	templateH := newTemplateHandler()

	// Templates (public) — витрина шаблонов доступна и до регистрации
	api.Get("/templates", templateH.getAll)
	api.Get("/templates/:id", templateH.getOne)

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
	guest.Get("/wishlists/:wishlistId/presents", presentH.getAll)
	guest.Put("/presents/:id/reserve", presentH.reserve)
	guest.Put("/presents/:id/release", presentH.release)

	// Protected routes
	protected := api.Group("")
	protected.Use(middleware.JWTProtected(jwtSecret))

	// Upload
	protected.Post("/upload", uploadH.upload)
	protected.Post("/upload/bulk", uploadH.bulkUpload)

	// Wishlists (protected)
	protected.Get("/wishlists", wishlistH.getAll)
	protected.Post("/wishlists", wishlistH.create)
	protected.Post("/wishlists/constructor", wishlistH.createConstructor)
	protected.Post("/wishlists/from-template", wishlistH.createFromTemplate)
	protected.Put("/wishlists/:id", wishlistH.update)
	protected.Put("/wishlists/:id/blocks", wishlistH.updateBlocks)
	protected.Delete("/wishlists/:id", wishlistH.delete)

	// Presents (protected)
	protected.Post("/wishlists/:wishlistId/presents", presentH.create)
	protected.Get("/presents/:id", presentH.getOne)
	protected.Put("/presents/:id", presentH.update)
	protected.Delete("/wishlists/:wishlistId/presents/:id", presentH.delete)
}
