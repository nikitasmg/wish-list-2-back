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
	parseUC usecase.ParseUseCase,
	templateUC usecase.TemplateUseCase,
) {
	api := router.Group("/api/v1")

	userH := newUserHandler(userUC, uploadUC, cookieDomain, secureCookie)
	wishlistH := newWishlistHandler(wishlistUC)
	presentH := newPresentHandler(presentUC)
	uploadH := newUploadHandler(uploadUC)
	parseH := newParseHandler(parseUC)
	templateH := newTemplateHandler(templateUC)

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

	// Templates (public) — BEFORE protected group
	api.Get("/templates", templateH.getPublic)

	// Wishlists (public) — static routes BEFORE parametric
	api.Get("/wishlists/s/:shortId", wishlistH.getByShortID)
	api.Get("/wishlists/:id", wishlistH.getOne)
	api.Get("/wishlists/:wishlistId/presents", presentH.getAll)
	api.Put("/presents/:id/reserve", presentH.reserve)
	api.Put("/presents/:id/release", presentH.release)

	// Protected routes
	protected := api.Group("")
	protected.Use(middleware.JWTProtected(jwtSecret))

	// User profile
	protected.Get("/users/me", userH.getProfile)
	protected.Patch("/users/me", userH.updateProfile)

	// Parse
	protected.Get("/parse", parseH.parse)

	// Upload
	protected.Post("/upload", uploadH.upload)
	protected.Post("/upload/bulk", uploadH.bulkUpload)

	// Wishlists (protected)
	protected.Get("/wishlists", wishlistH.getAll)
	protected.Post("/wishlists", wishlistH.create)
	protected.Post("/wishlists/constructor", wishlistH.createConstructor)
	protected.Put("/wishlists/:id", wishlistH.update)
	protected.Put("/wishlists/:id/blocks", wishlistH.updateBlocks)
	protected.Delete("/wishlists/:id", wishlistH.delete)

	// Presents (protected)
	protected.Post("/wishlists/:wishlistId/presents", presentH.create)
	protected.Get("/presents/:id", presentH.getOne)
	protected.Put("/presents/:id", presentH.update)
	protected.Delete("/wishlists/:wishlistId/presents/:id", presentH.delete)

	// Templates (protected)
	protected.Get("/templates/my", templateH.getMy)
	protected.Post("/templates", templateH.create)
	protected.Patch("/templates/:id", templateH.update)
	protected.Delete("/templates/:id", templateH.delete)
	protected.Post("/wishlists/from-template/:id", templateH.createWishlistFromTemplate)
}
