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
	"main/internal/entity"
	"main/internal/usecase"
)

func setupWishlistApp(wishlistMock usecase.WishlistUseCase) *fiber.App {
	app := fiber.New()
	userMock := &MockUserUC{}
	presentMock := &MockPresentUC{}
	uploadMock := &MockUploadUC{}
	v1.NewRouter(app, testSecret, "", false, userMock, wishlistMock, presentMock, uploadMock, &MockGuestDataUC{}, &MockTemplateUC{})
	return app
}

func TestCreate_MissingTitle(t *testing.T) {
	wm := &MockWishlistUC{}
	app := setupWishlistApp(wm)

	userID := uuid.New()
	token := makeTestToken(userID)

	// Send form without title
	body := bytes.NewBufferString("description=test")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wishlists", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Contains(t, result, "error")
}

func TestCreate_Success(t *testing.T) {
	wm := &MockWishlistUC{}
	app := setupWishlistApp(wm)

	userID := uuid.New()
	token := makeTestToken(userID)
	wid := uuid.New()

	wm.On("Create", mock.Anything, mock.Anything, mock.Anything).
		Return(entity.Wishlist{ID: wid, Title: "Test"}, nil)

	body := bytes.NewBufferString("title=Test+Wishlist")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wishlists", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Contains(t, result, "data")
}

func TestCreateConstructor_InvalidBlockType(t *testing.T) {
	wm := &MockWishlistUC{}
	app := setupWishlistApp(wm)

	userID := uuid.New()
	token := makeTestToken(userID)

	wm.On("CreateConstructor", mock.Anything, mock.Anything, mock.Anything).
		Return(entity.Wishlist{}, errors.New(`block[0]: unknown type "bad_type"`))

	payload := map[string]interface{}{
		"title":  "Test",
		"blocks": []map[string]interface{}{{"type": "bad_type", "position": 0}},
	}
	bodyBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wishlists/constructor", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Contains(t, result, "error")
}

func TestGetByShortID_NotFound(t *testing.T) {
	wm := &MockWishlistUC{}
	app := setupWishlistApp(wm)

	wm.On("GetByShortID", mock.Anything, "abc-def-ghi", mock.AnythingOfType("uuid.UUID")).
		Return(entity.Wishlist{}, errors.New("not found"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/wishlists/s/abc-def-ghi", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestGetByShortID_Success(t *testing.T) {
	wm := &MockWishlistUC{}
	app := setupWishlistApp(wm)

	wid := uuid.New()
	wm.On("GetByShortID", mock.Anything, "abc-def-ghi", mock.AnythingOfType("uuid.UUID")).
		Return(entity.Wishlist{ID: wid, ShortID: "abc-def-ghi", Title: "Test"}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/wishlists/s/abc-def-ghi", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	data, ok := result["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "abc-def-ghi", data["shortId"])
}

// PUT /blocks принимает и объект { blocks, rows } формата v3, и голый массив
// блоков от старых клиентов.
func TestUpdateBlocks_BodyFormats(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantRows []entity.RowSettings
	}{
		{name: "v3", body: `{"blocks":[{"id":"a","type":"text","row":0,"col":0,"colSpan":1}],"rows":[{"columns":3,"ratio":"1:1:1"}]}`,
			wantRows: []entity.RowSettings{{Columns: 3, Ratio: "1:1:1"}}},
		{name: "старый массив", body: `[{"id":"a","type":"text","row":0,"col":0,"colSpan":1}]`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wm := &MockWishlistUC{}
			app := setupWishlistApp(wm)
			userID, wid := uuid.New(), uuid.New()

			wm.On("UpdateBlocks", mock.Anything, userID, wid, mock.MatchedBy(func(b []entity.Block) bool {
				return len(b) == 1 && b[0].ID == "a"
			}), tc.wantRows, mock.Anything).Return(entity.Wishlist{ID: wid}, nil)

			req := httptest.NewRequest(http.MethodPut, "/api/v1/wishlists/"+wid.String()+"/blocks", bytes.NewBufferString(tc.body))
			req.Header.Set("Authorization", "Bearer "+makeTestToken(userID))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, fiber.StatusOK, resp.StatusCode)
			wm.AssertExpectations(t)
		})
	}
}
