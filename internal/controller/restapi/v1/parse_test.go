package v1_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"main/internal/entity"
	v1 "main/internal/controller/restapi/v1"
	parseUC "main/internal/usecase/parse"
)

func setupParseAppWithUC(pu *MockParseUC) *fiber.App {
	app := fiber.New()
	v1.NewRouter(app, testSecret, "localhost", false,
		&MockUserUC{}, &MockWishlistUC{}, &MockPresentUC{}, &MockUploadUC{}, pu, &MockTemplateUC{})
	return app
}

func doParseRequest(app *fiber.App, url string, userID uuid.UUID) *http.Response {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/parse?url="+url, nil)
	req.Header.Set("Authorization", "Bearer "+makeTestToken(userID))
	resp, _ := app.Test(req)
	return resp
}

func TestParse_MissingURL(t *testing.T) {
	app := setupParseAppWithUC(&MockParseUC{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/parse", nil)
	req.Header.Set("Authorization", "Bearer "+makeTestToken(uuid.New()))
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestParse_BadScheme(t *testing.T) {
	app := setupParseAppWithUC(&MockParseUC{})
	resp := doParseRequest(app, "ftp://ozon.ru/product/1", uuid.New())
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestParse_Success(t *testing.T) {
	pu := &MockParseUC{}
	userID := uuid.New()
	price := 8990.0
	pu.On("Parse", mock.Anything, userID, "https://ozon.ru/product/1").Return(entity.ParseResult{
		Title:    "Nike Air Max",
		Price:    &price,
		Source:   "ozon",
		ImageURL: "https://example.com/img.jpg",
	}, nil)

	app := setupParseAppWithUC(pu)
	resp := doParseRequest(app, "https://ozon.ru/product/1", userID)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "Nike Air Max", data["title"])
}

func TestParse_TitleEmpty_Returns422(t *testing.T) {
	pu := &MockParseUC{}
	userID := uuid.New()
	pu.On("Parse", mock.Anything, userID, "https://example.com").Return(entity.ParseResult{}, nil)

	app := setupParseAppWithUC(pu)
	resp := doParseRequest(app, "https://example.com", userID)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

func TestParse_RateLimit_Returns429(t *testing.T) {
	pu := &MockParseUC{}
	userID := uuid.New()
	pu.On("Parse", mock.Anything, userID, mock.Anything).Return(entity.ParseResult{}, parseUC.ErrRateLimit)

	app := setupParseAppWithUC(pu)
	resp := doParseRequest(app, "https://ozon.ru/product/1", userID)
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	assert.Equal(t, "3600", resp.Header.Get("Retry-After"))
}

func TestParse_Timeout_Returns504(t *testing.T) {
	pu := &MockParseUC{}
	userID := uuid.New()
	pu.On("Parse", mock.Anything, userID, mock.Anything).Return(entity.ParseResult{}, parseUC.ErrTimeout)

	app := setupParseAppWithUC(pu)
	resp := doParseRequest(app, "https://ozon.ru/product/1", userID)
	assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)
}

