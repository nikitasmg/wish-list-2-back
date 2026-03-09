package v1

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"

	"main/internal/controller/restapi/v1/request"
	"main/internal/controller/restapi/v1/response"
	"main/internal/usecase"
)

type userHandler struct {
	uc           usecase.UserUseCase
	cookieDomain string
}

func newUserHandler(uc usecase.UserUseCase, cookieDomain string) *userHandler {
	return &userHandler{uc: uc, cookieDomain: cookieDomain}
}

func (h *userHandler) register(c *fiber.Ctx) error {
	var req request.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid input"))
	}
	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("username and password are required"))
	}

	result, err := h.uc.Register(c.Context(), req.Username, req.Password)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}

	h.setTokenCookie(c, result.Token)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"token": result.Token})
}

func (h *userHandler) login(c *fiber.Ctx) error {
	var req request.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid input"))
	}

	result, err := h.uc.Login(c.Context(), req.Username, req.Password)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}

	h.setTokenCookie(c, result.Token)
	return c.JSON(fiber.Map{"token": result.Token})
}

func (h *userHandler) authTelegram(c *fiber.Ctx) error {
	var req request.TelegramAuthRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid input"))
	}

	result, err := h.uc.AuthenticateTelegram(c.Context(), usecase.TelegramAuthInput{
		ID:        req.ID,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		PhotoURL:  req.PhotoURL,
		Username:  req.Username,
		AuthDate:  req.AuthDate,
		Hash:      req.Hash,
	})
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}

	h.setTokenCookie(c, result.Token)
	return c.JSON(fiber.Map{"token": result.Token})
}

func (h *userHandler) me(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*jwt.Token)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error("invalid token"))
	}
	claims := user.Claims.(jwt.MapClaims)
	return c.JSON(fiber.Map{"user": claims})
}

func (h *userHandler) logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour * 24),
		Secure:   true,
		HTTPOnly: true,
		SameSite: fiber.CookieSameSiteLaxMode,
		Path:     "/",
		Domain:   h.cookieDomain,
	})
	return c.JSON(fiber.Map{"message": "logout successful"})
}

func (h *userHandler) setTokenCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    token,
		Expires:  time.Now().Add(time.Hour * 24 * 30),
		Secure:   true,
		HTTPOnly: true,
		SameSite: fiber.CookieSameSiteLaxMode,
		Path:     "/",
		Domain:   h.cookieDomain,
	})
}
