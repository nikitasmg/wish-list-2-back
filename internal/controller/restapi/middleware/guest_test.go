package middleware_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"main/internal/controller/restapi/middleware"
)

const secret = "test-secret"

// newGuestApp поднимает приложение, которое отдаёт распознанного гостя текстом.
func newGuestApp() *fiber.App {
	app := fiber.New()
	app.Use(middleware.GuestIdentity(secret, "", false))
	app.Get("/", func(c *fiber.Ctx) error {
		id, ok := middleware.GuestIDFromCtx(c)
		if !ok {
			return c.SendString("none")
		}
		return c.SendString(id.String())
	})
	return app
}

// guestCookie вытаскивает значение выставленной куки из ответа.
func guestCookie(t *testing.T, app *fiber.App, cookie string) (body string, setCookie string) {
	t.Helper()
	req := httptest.NewRequest("GET", "/", nil)
	if cookie != "" {
		req.Header.Set("Cookie", middleware.GuestCookieName+"="+cookie)
	}
	resp, err := app.Test(req)
	require.NoError(t, err)

	buf := make([]byte, resp.ContentLength)
	_, _ = resp.Body.Read(buf)

	for _, c := range resp.Cookies() {
		if c.Name == middleware.GuestCookieName {
			setCookie = c.Value
		}
	}
	return string(buf), setCookie
}

func TestGuestIdentity_IssuesCookieOnFirstVisit(t *testing.T) {
	app := newGuestApp()

	body, cookie := guestCookie(t, app, "")

	require.NotEmpty(t, cookie, "кука должна выставляться при первом обращении")
	_, err := uuid.Parse(body)
	assert.NoError(t, err, "в контекст должен попасть валидный UUID")
	assert.Contains(t, cookie, body, "кука подписывает тот же id, что ушёл в контекст")
}

func TestGuestIdentity_KeepsSameGuestAcrossRequests(t *testing.T) {
	app := newGuestApp()

	first, cookie := guestCookie(t, app, "")
	second, reissued := guestCookie(t, app, cookie)

	assert.Equal(t, first, second, "тот же браузер — тот же гость")
	assert.Empty(t, reissued, "валидную куку перевыставлять не нужно")
}

func TestGuestIdentity_RejectsForgedCookie(t *testing.T) {
	app := newGuestApp()
	forged := uuid.New().String() + ".not-a-real-signature"

	body, reissued := guestCookie(t, app, forged)

	assert.NotEqual(t, forged, body)
	require.NotEmpty(t, reissued, "подделанная кука заменяется новой")
	assert.NotContains(t, reissued, "not-a-real-signature")
}

func TestGuestIdentity_RejectsCookieSignedWithAnotherSecret(t *testing.T) {
	issuer := fiber.New()
	issuer.Use(middleware.GuestIdentity("another-secret", "", false))
	issuer.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })
	_, foreign := guestCookie(t, issuer, "")
	require.NotEmpty(t, foreign)

	app := newGuestApp()
	body, reissued := guestCookie(t, app, foreign)

	assert.NotContains(t, foreign, body, "чужая подпись не принимается")
	assert.NotEmpty(t, reissued)
}
