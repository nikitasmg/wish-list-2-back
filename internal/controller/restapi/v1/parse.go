package v1

import (
	"errors"
	"net/url"

	"github.com/gofiber/fiber/v2"

	"main/internal/controller/restapi/v1/response"
	"main/internal/usecase"
	parseUC "main/internal/usecase/parse"
)

type parseHandler struct {
	uc usecase.ParseUseCase
}

func newParseHandler(uc usecase.ParseUseCase) *parseHandler {
	return &parseHandler{uc: uc}
}

func (h *parseHandler) parse(c *fiber.Ctx) error {
	rawURL := c.Query("url")
	if rawURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("url is required"))
	}

	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid url: must be http or https"))
	}

	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}

	result, err := h.uc.Parse(c.Context(), userID, rawURL)
	if err != nil {
		switch {
		case errors.Is(err, parseUC.ErrRateLimit):
			c.Set("Retry-After", "3600")
			return c.Status(fiber.StatusTooManyRequests).JSON(response.Error("rate limit exceeded"))
		case errors.Is(err, parseUC.ErrTimeout):
			return c.Status(fiber.StatusGatewayTimeout).JSON(response.Error("parse timeout"))
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
		}
	}

	if result.Title == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("could not parse title from page"))
	}

	return c.JSON(response.Data(result))
}
