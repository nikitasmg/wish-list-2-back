package response

import "github.com/gofiber/fiber/v2"

func Data(v interface{}) fiber.Map {
	return fiber.Map{"data": v}
}

func Error(msg string) fiber.Map {
	return fiber.Map{"error": msg}
}
