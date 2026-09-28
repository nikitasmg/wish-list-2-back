package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	// GuestCookieName — кука, по которой узнаётся гость без регистрации
	GuestCookieName = "guest_id"
	// GuestContextKey — ключ в c.Locals, куда кладётся распознанный гость
	GuestContextKey = "guestID"

	guestCookieTTL = 365 * 24 * time.Hour
)

// GuestIdentity опознаёт гостя без регистрации: подписанный UUID в куке.
// Кука ставится при первом обращении к публичной странице и живёт год, поэтому
// один и тот же человек узнаётся между визитами — этого хватает, чтобы он мог
// снять свою бронь, проголосовать один раз и увидеть собственный ответ.
//
// Подпись нужна, чтобы гость не мог подставить чужой идентификатор и снять
// чужую бронь: значение куки проверяется тем же секретом, что и JWT.
func GuestIdentity(secret, cookieDomain string, secureCookie bool) fiber.Handler {
	key := []byte(secret)

	return func(c *fiber.Ctx) error {
		id, ok := parseGuestCookie(c.Cookies(GuestCookieName), key)
		if !ok {
			id = uuid.New()
			c.Cookie(&fiber.Cookie{
				Name:     GuestCookieName,
				Value:    signGuestID(id, key),
				Path:     "/",
				Domain:   cookieDomain,
				Expires:  time.Now().Add(guestCookieTTL),
				HTTPOnly: true,
				Secure:   secureCookie,
				SameSite: "Lax",
			})
		}

		c.Locals(GuestContextKey, id)
		return c.Next()
	}
}

// GuestIDFromCtx достаёт гостя, положенного GuestIdentity.
func GuestIDFromCtx(c *fiber.Ctx) (uuid.UUID, bool) {
	id, ok := c.Locals(GuestContextKey).(uuid.UUID)
	if !ok || id == uuid.Nil {
		return uuid.Nil, false
	}
	return id, true
}

func signGuestID(id uuid.UUID, key []byte) string {
	raw := id.String()
	return raw + "." + guestSignature(raw, key)
}

func parseGuestCookie(value string, key []byte) (uuid.UUID, bool) {
	raw, signature, found := strings.Cut(value, ".")
	if !found {
		return uuid.Nil, false
	}
	if !hmac.Equal([]byte(signature), []byte(guestSignature(raw, key))) {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func guestSignature(raw string, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
