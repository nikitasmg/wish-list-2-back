package v1_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1 "main/internal/controller/restapi/v1"
	"main/internal/entity"
	"main/internal/usecase"
)

func setupTemplateApp(tm *MockTemplateUC) *fiber.App {
	app := fiber.New()
	v1.NewRouter(app, testSecret, "", false,
		&MockUserUC{}, &MockWishlistUC{}, &MockPresentUC{}, &MockUploadUC{}, &MockParseUC{},
		tm,
	)
	return app
}

func TestGetPublicTemplates_Success(t *testing.T) {
	tm := &MockTemplateUC{}
	app := setupTemplateApp(tm)

	templates := []entity.TemplateWithAuthor{
		{
			Template: entity.Template{
				ID:        uuid.New(),
				Name:      "Birthday template",
				IsPublic:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			UserDisplayName: "Никита",
		},
	}
	tm.On("GetPublic", mock.Anything, 0, "").Return(templates, "", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	data := result["data"].([]interface{})
	assert.Len(t, data, 1)
}

func TestCreateTemplate_Success(t *testing.T) {
	tm := &MockTemplateUC{}
	app := setupTemplateApp(tm)

	userID := uuid.New()
	token := makeTestToken(userID)
	wishlistID := uuid.New()

	created := entity.Template{
		ID:       uuid.New(),
		UserID:   userID,
		Name:     "Мой шаблон",
		IsPublic: false,
	}
	tm.On("Create", mock.Anything, userID, usecase.CreateTemplateInput{
		WishlistID: wishlistID,
		Name:       "Мой шаблон",
		IsPublic:   false,
	}).Return(created, nil)

	body := bytes.NewBufferString(`{"wishlistId":"` + wishlistID.String() + `","name":"Мой шаблон","isPublic":false}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
}

func TestCreateTemplate_Unauthorized(t *testing.T) {
	tm := &MockTemplateUC{}
	app := setupTemplateApp(tm)

	body := bytes.NewBufferString(`{"wishlistId":"` + uuid.New().String() + `","name":"test","isPublic":false}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", body)
	req.Header.Set("Content-Type", "application/json")
	// No auth token

	resp, err := app.Test(req)
	require.NoError(t, err)
	// gofiber/jwt middleware returns 400 when Authorization header is absent
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestDeleteTemplate_Forbidden(t *testing.T) {
	tm := &MockTemplateUC{}
	app := setupTemplateApp(tm)

	userID := uuid.New()
	token := makeTestToken(userID)
	templateID := uuid.New()

	tm.On("Delete", mock.Anything, templateID, userID).Return(errors.New("forbidden"))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/templates/"+templateID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestCreateWishlistFromTemplate_Success(t *testing.T) {
	tm := &MockTemplateUC{}
	app := setupTemplateApp(tm)

	userID := uuid.New()
	token := makeTestToken(userID)
	templateID := uuid.New()

	wishlist := entity.Wishlist{ID: uuid.New(), Title: "Мой день рождения"}
	tm.On("CreateWishlistFromTemplate", mock.Anything, templateID, userID, "Мой день рождения").
		Return(wishlist, nil)

	body := bytes.NewBufferString(`{"title":"Мой день рождения"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wishlists/from-template/"+templateID.String(), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
}
