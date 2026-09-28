package v1

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

	"main/internal/controller/restapi/v1/response"
	"main/internal/usecase"
)

func getUserID(c *fiber.Ctx) (uuid.UUID, error) {
	user, ok := c.Locals("user").(*jwt.Token)
	if !ok {
		return uuid.UUID{}, errors.New("could not parse token")
	}
	claims, ok := user.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.UUID{}, errors.New("could not parse claims")
	}
	idStr, ok := claims["id"].(string)
	if !ok {
		return uuid.UUID{}, errors.New("id not found in claims")
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.UUID{}, errors.New("invalid UUID format")
	}
	return id, nil
}

func stringToBool(s string) bool {
	return s == "true" || s == "1"
}

// ownerError переводит отказ по правам в 403, всё остальное — в переданный
// статус. Без отдельной ветки чужой вишлист отвечал бы «500 internal error»,
// хотя запрос отработал ровно так, как должен.
func ownerError(c *fiber.Ctx, err error, fallback int) error {
	if errors.Is(err, usecase.ErrForbidden) {
		return c.Status(fiber.StatusForbidden).JSON(response.Error(err.Error()))
	}
	return c.Status(fallback).JSON(response.Error(err.Error()))
}
