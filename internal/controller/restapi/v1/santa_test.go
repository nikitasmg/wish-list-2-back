package v1_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"main/internal/entity"
	v1 "main/internal/controller/restapi/v1"
	"main/internal/usecase"
)

type MockSantaUC struct{ mock.Mock }

func (m *MockSantaUC) CreateRoom(ctx context.Context, ownerID uuid.UUID, in usecase.SantaRoomInput) (entity.SantaRoom, error) {
	args := m.Called(ctx, ownerID, in)
	return args.Get(0).(entity.SantaRoom), args.Error(1)
}
func (m *MockSantaUC) ListRooms(ctx context.Context, userID uuid.UUID) ([]usecase.SantaRoomSummary, error) {
	args := m.Called(ctx, userID)
	rooms, _ := args.Get(0).([]usecase.SantaRoomSummary)
	return rooms, args.Error(1)
}
func (m *MockSantaUC) GetRoom(ctx context.Context, ownerID, roomID uuid.UUID) (usecase.SantaRoomDetails, error) {
	args := m.Called(ctx, ownerID, roomID)
	return args.Get(0).(usecase.SantaRoomDetails), args.Error(1)
}
func (m *MockSantaUC) UpdateRoom(ctx context.Context, ownerID, roomID uuid.UUID, in usecase.SantaRoomInput) (entity.SantaRoom, error) {
	args := m.Called(ctx, ownerID, roomID, in)
	return args.Get(0).(entity.SantaRoom), args.Error(1)
}
func (m *MockSantaUC) DeleteRoom(ctx context.Context, ownerID, roomID uuid.UUID) error {
	return m.Called(ctx, ownerID, roomID).Error(0)
}
func (m *MockSantaUC) RemoveParticipant(ctx context.Context, ownerID, roomID, participantID uuid.UUID) error {
	return m.Called(ctx, ownerID, roomID, participantID).Error(0)
}
func (m *MockSantaUC) GetInvite(ctx context.Context, slug string) (usecase.SantaInvite, error) {
	args := m.Called(ctx, slug)
	return args.Get(0).(usecase.SantaInvite), args.Error(1)
}
func (m *MockSantaUC) Join(ctx context.Context, slug string, userID *uuid.UUID, in usecase.SantaProfileInput) (usecase.SantaJoinResult, error) {
	args := m.Called(ctx, slug, userID, in)
	return args.Get(0).(usecase.SantaJoinResult), args.Error(1)
}
func (m *MockSantaUC) GetMe(ctx context.Context, slug string, auth usecase.SantaAuth) (usecase.SantaMe, error) {
	args := m.Called(ctx, slug, auth)
	return args.Get(0).(usecase.SantaMe), args.Error(1)
}
func (m *MockSantaUC) UpdateMe(ctx context.Context, slug string, auth usecase.SantaAuth, in usecase.SantaProfileInput) (usecase.SantaMe, error) {
	args := m.Called(ctx, slug, auth, in)
	return args.Get(0).(usecase.SantaMe), args.Error(1)
}
func (m *MockSantaUC) LeaveMe(ctx context.Context, slug string, auth usecase.SantaAuth) error {
	return m.Called(ctx, slug, auth).Error(0)
}
func (m *MockSantaUC) Draw(ctx context.Context, ownerID, roomID uuid.UUID) error {
	return m.Called(ctx, ownerID, roomID).Error(0)
}
func (m *MockSantaUC) Redraw(ctx context.Context, ownerID, roomID uuid.UUID) error {
	return m.Called(ctx, ownerID, roomID).Error(0)
}

func newSantaApp(m *MockSantaUC) *fiber.App {
	app := fiber.New()
	v1.NewSantaRouter(app, testSecret, m)
	return app
}

func doReq(t *testing.T, app *fiber.App, req *http.Request) (int, string) {
	t.Helper()
	resp, err := app.Test(req)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func TestSantaInvite_NotFound(t *testing.T) {
	m := &MockSantaUC{}
	m.On("GetInvite", mock.Anything, "nope").Return(usecase.SantaInvite{}, usecase.ErrSantaNotFound)

	status, _ := doReq(t, newSantaApp(m), httptest.NewRequest(http.MethodGet, "/api/v1/santa/r/nope", nil))

	assert.Equal(t, http.StatusNotFound, status)
}

func TestSantaRooms_RequireLogin(t *testing.T) {
	m := &MockSantaUC{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/santa/rooms", strings.NewReader(`{"title":"Офис"}`))
	req.Header.Set("Content-Type", "application/json")

	status, _ := doReq(t, newSantaApp(m), req)

	assert.Equal(t, http.StatusUnauthorized, status)
	m.AssertNotCalled(t, "CreateRoom", mock.Anything, mock.Anything, mock.Anything)
}

func TestSantaCreateRoom_ParsesDates(t *testing.T) {
	m := &MockSantaUC{}
	user := uuid.New()
	m.On("CreateRoom", mock.Anything, user, mock.MatchedBy(func(in usecase.SantaRoomInput) bool {
		return in.Title == "Офис" && in.ExchangeDate != nil && in.ExchangeDate.Format("2006-01-02") == "2026-12-27" &&
			in.Budget != nil && *in.Budget == 3000 && in.OrganizerJoins
	})).Return(entity.SantaRoom{ID: uuid.New(), Title: "Офис"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/santa/rooms",
		strings.NewReader(`{"title":"Офис","budget":3000,"exchangeDate":"2026-12-27","organizerJoins":true,"organizerName":"Никита"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+makeTestToken(user))

	status, _ := doReq(t, newSantaApp(m), req)

	assert.Equal(t, http.StatusCreated, status)
	m.AssertExpectations(t)
}

func TestSantaCreateRoom_BadDateIs422(t *testing.T) {
	m := &MockSantaUC{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/santa/rooms", strings.NewReader(`{"title":"Офис","exchangeDate":"27.12.2026"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+makeTestToken(uuid.New()))

	status, _ := doReq(t, newSantaApp(m), req)

	assert.Equal(t, http.StatusUnprocessableEntity, status)
}

func TestSantaDraw_TooFewIs422(t *testing.T) {
	m := &MockSantaUC{}
	user, room := uuid.New(), uuid.New()
	m.On("Draw", mock.Anything, user, room).Return(usecase.ErrSantaTooFew)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/santa/rooms/"+room.String()+"/draw", nil)
	req.Header.Set("Authorization", "Bearer "+makeTestToken(user))

	status, body := doReq(t, newSantaApp(m), req)

	assert.Equal(t, http.StatusUnprocessableEntity, status)
	assert.Contains(t, body, "минимум 3")
}

func TestSantaDraw_RepeatIs409(t *testing.T) {
	m := &MockSantaUC{}
	user, room := uuid.New(), uuid.New()
	m.On("Draw", mock.Anything, user, room).Return(usecase.ErrSantaDrawn)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/santa/rooms/"+room.String()+"/draw", nil)
	req.Header.Set("Authorization", "Bearer "+makeTestToken(user))

	status, _ := doReq(t, newSantaApp(m), req)

	assert.Equal(t, http.StatusConflict, status)
}

func TestSantaMe_PassesTokenHeader(t *testing.T) {
	m := &MockSantaUC{}
	m.On("GetMe", mock.Anything, "AbCd2345", usecase.SantaAuth{Token: "tok"}).
		Return(usecase.SantaMe{Name: "Маша"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/santa/r/AbCd2345/me", nil)
	req.Header.Set(v1.SantaTokenHeader, "tok")

	status, body := doReq(t, newSantaApp(m), req)

	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, `"name":"Маша"`)
}

func TestSantaJoin_LoggedInUserIsPassed(t *testing.T) {
	m := &MockSantaUC{}
	user := uuid.New()
	m.On("Join", mock.Anything, "AbCd2345", mock.MatchedBy(func(id *uuid.UUID) bool { return id != nil && *id == user }),
		usecase.SantaProfileInput{Name: "Маша", Wishes: "чай", WishlistURL: ""}).
		Return(usecase.SantaJoinResult{Token: "tok"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/santa/r/AbCd2345/join", strings.NewReader(`{"name":"Маша","wishes":"чай"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+makeTestToken(user))

	status, body := doReq(t, newSantaApp(m), req)

	assert.Equal(t, http.StatusCreated, status)
	assert.Contains(t, body, `"token":"tok"`)
}

// Публичные маршруты Санты зарегистрированы до основного роутера, иначе его
// защищённая группа с префиксом "" требовала бы вход и на них.
func TestSantaRoutes_PublicBeforeProtectedGroup(t *testing.T) {
	m := &MockSantaUC{}
	m.On("GetInvite", mock.Anything, "AbCd2345").Return(usecase.SantaInvite{Title: "Офис"}, nil)
	app := fiber.New()
	v1.NewSantaRouter(app, testSecret, m)
	v1.NewRouter(app, testSecret, "", false, &MockUserUC{}, &MockWishlistUC{}, &MockPresentUC{}, &MockUploadUC{}, &MockGuestDataUC{}, &MockTemplateUC{})

	status, _ := doReq(t, app, httptest.NewRequest(http.MethodGet, "/api/v1/santa/r/AbCd2345", nil))

	assert.Equal(t, http.StatusOK, status)
}

func TestSantaError_WrappedNotFoundHidesPrefix(t *testing.T) {
	m := &MockSantaUC{}
	m.On("GetInvite", mock.Anything, "x").Return(usecase.SantaInvite{}, fmt.Errorf("receiver: %w", usecase.ErrSantaNotFound))

	status, body := doReq(t, newSantaApp(m), httptest.NewRequest(http.MethodGet, "/api/v1/santa/r/x", nil))

	assert.Equal(t, http.StatusNotFound, status)
	assert.JSONEq(t, `{"error":"`+usecase.ErrSantaNotFound.Error()+`"}`, body)
}
