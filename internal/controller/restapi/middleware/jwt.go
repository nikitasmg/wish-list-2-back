package middleware

import (
	"github.com/gofiber/fiber/v2"
	jwtware "github.com/gofiber/jwt/v2"
)

// CookieToHeader переносит JWT-токен из cookie в заголовок Authorization
func CookieToHeader() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Cookies("token")
		if token != "" {
			c.Request().Header.Set("Authorization", "Bearer "+token)
		}
		return c.Next()
	}
}

// JWTProtected создаёт middleware для проверки JWT
func JWTProtected(secret string) fiber.Handler {
	return jwtware.New(jwtware.Config{
		SigningKey: []byte(secret),
		ContextKey: "user",
	})
}

// JWTOptional parses JWT if present but does not fail on missing/invalid tokens.
// Use for routes that have optional authentication (public + enriched for logged-in users).
func JWTOptional(secret string) fiber.Handler {
	return jwtware.New(jwtware.Config{
		SigningKey:  []byte(secret),
		ContextKey:  "user",
		ErrorHandler: func(c *fiber.Ctx, _ error) error {
			return c.Next()
		},
	})
}
