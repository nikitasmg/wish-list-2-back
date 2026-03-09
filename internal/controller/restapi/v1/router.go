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
	userUC usecase.UserUseCase,
	wishlistUC usecase.WishlistUseCase,
	presentUC usecase.PresentUseCase,
) {
	api := router.Group("/api/v1")

	userH := newUserHandler(userUC, cookieDomain)
	wishlistH := newWishlistHandler(wishlistUC)
	presentH := newPresentHandler(presentUC)

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

	// Wishlists (public)
	api.Get("/wishlists/:id", wishlistH.getOne)
	api.Get("/wishlists/:wishlistId/presents", presentH.getAll)
	api.Put("/presents/:id/reserve", presentH.reserve)
	api.Put("/presents/:id/release", presentH.release)

	// Wishlists (protected)
	protected := api.Group("")
	protected.Use(middleware.JWTProtected(jwtSecret))
	protected.Get("/wishlists", wishlistH.getAll)
	protected.Post("/wishlists", wishlistH.create)
	protected.Put("/wishlists/:id", wishlistH.update)
	protected.Delete("/wishlists/:id", wishlistH.delete)
	protected.Post("/wishlists/:wishlistId/presents", presentH.create)
	protected.Get("/presents/:id", presentH.getOne)
	protected.Put("/presents/:id", presentH.update)
	protected.Delete("/wishlists/:wishlistId/presents/:id", presentH.delete)
}
