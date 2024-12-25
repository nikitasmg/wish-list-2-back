package helpers

import (
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

func GetUserId(c *fiber.Ctx) (error, *uuid.UUID) {
	user, ok := c.Locals("user").(*jwt.Token) // Получаем токен из контекста
	if !ok {
		return errors.New("Could not parse token"), nil
	}

	claims, ok := user.Claims.(jwt.MapClaims) // Приведение claims к корректному типу
	if !ok {
		return errors.New("Could not parse claims"), nil
	}
	idStr, ok := claims["id"].(string) // Получаем ID пользователя как строку
	if !ok {
		return errors.New("ID not found in claims"), nil
	}

	id, err := uuid.Parse(idStr) // Преобразуем строку в uuid.UUID
	if err != nil {
		return errors.New("invalid UUID format"), nil
	}

	return nil, &id // Возвращаем указатель на ID пользователя
}
