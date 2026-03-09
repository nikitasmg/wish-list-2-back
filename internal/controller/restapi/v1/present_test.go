package v1_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1 "main/internal/controller/restapi/v1"
	"main/internal/usecase"
)

func setupPresentApp(presentMock usecase.PresentUseCase) *fiber.App {
	app := fiber.New()
	userMock := &MockUserUC{}
	wishlistMock := &MockWishlistUC{}
	uploadMock := &MockUploadUC{}
	v1.NewRouter(app, testSecret, "", userMock, wishlistMock, presentMock, uploadMock)
	return app
}

func TestReserve_AlreadyReserved(t *testing.T) {
	pm := &MockPresentUC{}
	app := setupPresentApp(pm)

	pid := uuid.New()
	pm.On("Reserve", mock.Anything, pid).Return(errors.New("упс... подарок уже был забронирован, пожалуйста перезагрузите страницу"))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/presents/"+pid.String()+"/reserve", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Contains(t, result["error"].(string), "забронирован")
}

func TestReserve_Success(t *testing.T) {
	pm := &MockPresentUC{}
	app := setupPresentApp(pm)

	pid := uuid.New()
	pm.On("Reserve", mock.Anything, pid).Return(nil)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/presents/"+pid.String()+"/reserve", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, true, result["data"])
}

func TestCreate_InvalidWishlistID(t *testing.T) {
	pm := &MockPresentUC{}
	app := setupPresentApp(pm)

	userID := uuid.New()
	token := makeTestToken(userID)

	body := bytes.NewBufferString("title=Gift")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wishlists/not-a-uuid/presents", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestDelete_Success(t *testing.T) {
	pm := &MockPresentUC{}
	app := setupPresentApp(pm)

	userID := uuid.New()
	token := makeTestToken(userID)
	pid := uuid.New()
	wid := uuid.New()

	pm.On("Delete", mock.Anything, wid, pid).Return(nil)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/wishlists/"+wid.String()+"/presents/"+pid.String(),
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, true, result["data"])
}
