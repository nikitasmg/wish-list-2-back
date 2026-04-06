package v1

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
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

func getOptionalUserID(c *fiber.Ctx) *uuid.UUID {
	id, err := getUserID(c)
	if err != nil {
		return nil
	}
	return &id
}
