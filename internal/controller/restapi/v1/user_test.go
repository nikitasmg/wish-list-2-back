package v1_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1 "main/internal/controller/restapi/v1"
	"main/internal/entity"
	"main/internal/usecase"
)

func setupUserApp(userMock usecase.UserUseCase) *fiber.App {
	app := fiber.New()
	wishlistMock := &MockWishlistUC{}
	presentMock := &MockPresentUC{}
	uploadMock := &MockUploadUC{}
	v1.NewRouter(app, testSecret, "", false, userMock, wishlistMock, presentMock, uploadMock, &MockParseUC{}, &MockTemplateUC{})
	return app
}

func TestRegister_Success(t *testing.T) {
	um := &MockUserUC{}
	app := setupUserApp(um)

	userID := uuid.New()
	um.On("Register", mock.Anything, "alice", "password123").Return(usecase.AuthResult{
		Token: makeTestToken(userID),
		User:  entity.User{ID: userID, Username: "alice"},
	}, nil)

	body := bytes.NewBufferString(`{"username":"alice","password":"password123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	// Cookie "token" should be set
	var tokenCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "token" {
			tokenCookie = c
			break
		}
	}
	require.NotNil(t, tokenCookie, "cookie 'token' should be set")
	assert.NotEmpty(t, tokenCookie.Value)
}

func TestLogin_Unauthorized(t *testing.T) {
	um := &MockUserUC{}
	app := setupUserApp(um)

	um.On("Login", mock.Anything, "alice", "wrong").Return(
		usecase.AuthResult{},
		errors.New("неверный логин или пароль"),
	)

	body := bytes.NewBufferString(`{"username":"alice","password":"wrong"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "неверный логин или пароль", result["error"].(string))
}

func TestUploadBulk_NoFiles(t *testing.T) {
	um := &MockUserUC{}
	app := setupUserApp(um)

	userID := uuid.New()
	token := makeTestToken(userID)

	// Create a multipart form with no "files" field — only a text field
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.WriteField("other_field", "value")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload/bulk", &buf)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Contains(t, result["error"].(string), "files are required")
}
