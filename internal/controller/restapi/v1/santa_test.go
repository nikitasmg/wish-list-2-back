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

	v1 "main/internal/controller/restapi/v1"
	"main/internal/entity"
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
func (m *MockSantaUC) Join(ctx context.Context, slug string, auth usecase.SantaAuth, in usecase.SantaProfileInput) (usecase.SantaJoinResult, error) {
	args := m.Called(ctx, slug, auth, in)
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
	v1.NewSantaRouter(app, testSecret, "hook-secret", m)
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
	m.On("Join", mock.Anything, "AbCd2345", mock.MatchedBy(func(a usecase.SantaAuth) bool { return a.UserID != nil && *a.UserID == user }),
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
	v1.NewSantaRouter(app, testSecret, "hook-secret", m)
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

func TestSantaJoin_PassesTokenHeader(t *testing.T) {
	m := &MockSantaUC{}
	m.On("Join", mock.Anything, "AbCd2345", usecase.SantaAuth{Token: "tok"}, usecase.SantaProfileInput{Name: "Маша"}).
		Return(usecase.SantaJoinResult{}, usecase.ErrSantaAlreadyJoined)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/santa/r/AbCd2345/join", strings.NewReader(`{"name":"Маша"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(v1.SantaTokenHeader, "tok")

	status, _ := doReq(t, newSantaApp(m), req)

	assert.Equal(t, http.StatusConflict, status)
	m.AssertExpectations(t)
}

func (m *MockSantaUC) Remind(ctx context.Context, ownerID, roomID uuid.UUID) (usecase.SantaRemindResult, error) {
	args := m.Called(ctx, ownerID, roomID)
	r, _ := args.Get(0).(usecase.SantaRemindResult)
	return r, args.Error(1)
}
func (m *MockSantaUC) RequestEmailCode(ctx context.Context, slug string, auth usecase.SantaAuth, email string) error {
	return m.Called(ctx, slug, auth, email).Error(0)
}
func (m *MockSantaUC) VerifyEmail(ctx context.Context, slug string, auth usecase.SantaAuth, code string) (usecase.SantaMe, error) {
	args := m.Called(ctx, slug, auth, code)
	me, _ := args.Get(0).(usecase.SantaMe)
	return me, args.Error(1)
}
func (m *MockSantaUC) TelegramLink(ctx context.Context, slug string, auth usecase.SantaAuth) (string, error) {
	args := m.Called(ctx, slug, auth)
	return args.String(0), args.Error(1)
}
func (m *MockSantaUC) TelegramStart(ctx context.Context, chatID int64, token string) error {
	return m.Called(ctx, chatID, token).Error(0)
}
func (m *MockSantaUC) GetChat(ctx context.Context, slug string, auth usecase.SantaAuth, with usecase.SantaChatWith) (usecase.SantaChat, error) {
	args := m.Called(ctx, slug, auth, with)
	return args.Get(0).(usecase.SantaChat), args.Error(1)
}
func (m *MockSantaUC) SendChat(ctx context.Context, slug string, auth usecase.SantaAuth, with usecase.SantaChatWith, body string) (usecase.SantaChatMessage, error) {
	args := m.Called(ctx, slug, auth, with, body)
	return args.Get(0).(usecase.SantaChatMessage), args.Error(1)
}
func (m *MockSantaUC) TelegramReply(ctx context.Context, chatID, replyToMessageID int64, text string) error {
	return m.Called(ctx, chatID, replyToMessageID, text).Error(0)
}

func jsonReq(path, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestSantaRequestEmailCode(t *testing.T) {
	m := &MockSantaUC{}
	m.On("RequestEmailCode", mock.Anything, "abcdefgh", usecase.SantaAuth{Token: "tok"}, "a@example.com").Return(nil)
	req := jsonReq("/api/v1/santa/r/abcdefgh/me/email", `{"email":"a@example.com"}`)
	req.Header.Set(v1.SantaTokenHeader, "tok")

	status, _ := doReq(t, newSantaApp(m), req)

	assert.Equal(t, http.StatusOK, status)
	m.AssertExpectations(t)
}

func TestSantaRequestEmailCode_TooSoonIs429(t *testing.T) {
	m := &MockSantaUC{}
	m.On("RequestEmailCode", mock.Anything, "abcdefgh", mock.Anything, mock.Anything).Return(usecase.ErrSantaTooSoon)

	status, _ := doReq(t, newSantaApp(m), jsonReq("/api/v1/santa/r/abcdefgh/me/email", `{"email":"a@example.com"}`))

	assert.Equal(t, http.StatusTooManyRequests, status)
}

func TestSantaEmailTakenIs409(t *testing.T) {
	m := &MockSantaUC{}
	m.On("RequestEmailCode", mock.Anything, "abcdefgh", mock.Anything, mock.Anything).Return(usecase.ErrSantaEmailTaken)

	status, _ := doReq(t, newSantaApp(m), jsonReq("/api/v1/santa/r/abcdefgh/me/email", `{"email":"a@example.com"}`))

	assert.Equal(t, http.StatusConflict, status)
}

func TestSantaVerifyEmail(t *testing.T) {
	m := &MockSantaUC{}
	m.On("VerifyEmail", mock.Anything, "abcdefgh", mock.Anything, "123456").
		Return(usecase.SantaMe{Name: "Аня", Notify: usecase.SantaNotifyView{Ready: true}}, nil)

	status, body := doReq(t, newSantaApp(m), jsonReq("/api/v1/santa/r/abcdefgh/me/email/verify", `{"code":"123456"}`))

	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, `"ready":true`)
}

func TestSantaTelegramLink(t *testing.T) {
	m := &MockSantaUC{}
	m.On("TelegramLink", mock.Anything, "abcdefgh", mock.Anything).Return("https://t.me/bot?start=x", nil)

	status, body := doReq(t, newSantaApp(m), httptest.NewRequest(http.MethodPost, "/api/v1/santa/r/abcdefgh/me/telegram", nil))

	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"data":{"url":"https://t.me/bot?start=x"}}`, body)
}

func TestTelegramWebhook_StartWithToken(t *testing.T) {
	m := &MockSantaUC{}
	m.On("TelegramStart", mock.Anything, int64(77), "RAWTOKEN").Return(nil)
	req := jsonReq("/api/v1/telegram/webhook", `{"update_id":1,"message":{"chat":{"id":77},"text":"/start RAWTOKEN"}}`)
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "hook-secret")

	status, _ := doReq(t, newSantaApp(m), req)

	assert.Equal(t, http.StatusOK, status)
	m.AssertExpectations(t)
}

func TestTelegramWebhook_RejectsWrongSecret(t *testing.T) {
	m := &MockSantaUC{}
	req := jsonReq("/api/v1/telegram/webhook", `{"message":{"chat":{"id":77},"text":"/start X"}}`)
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong")

	status, _ := doReq(t, newSantaApp(m), req)

	assert.Equal(t, http.StatusUnauthorized, status)
	m.AssertNotCalled(t, "TelegramStart", mock.Anything, mock.Anything, mock.Anything)
}

func TestTelegramWebhook_IgnoresOtherUpdates(t *testing.T) {
	m := &MockSantaUC{}
	app := newSantaApp(m)
	for _, body := range []string{`{"edited_message":{}}`, `{"message":{"chat":{"id":1},"text":"привет"}}`, `не json`} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/telegram/webhook", strings.NewReader(body))
		req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "hook-secret")
		status, _ := doReq(t, app, req)
		assert.Equal(t, http.StatusOK, status, body)
	}
	m.AssertNotCalled(t, "TelegramStart", mock.Anything, mock.Anything, mock.Anything)
	m.AssertNotCalled(t, "TelegramReply", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestTelegramWebhook_DisabledWithoutSecret(t *testing.T) {
	app := fiber.New()
	v1.NewSantaRouter(app, testSecret, "", &MockSantaUC{})

	status, _ := doReq(t, app, jsonReq("/api/v1/telegram/webhook", `{}`))

	assert.Equal(t, http.StatusNotFound, status)
}

func TestSantaRemind(t *testing.T) {
	m := &MockSantaUC{}
	user, roomID := uuid.New(), uuid.New()
	m.On("Remind", mock.Anything, user, roomID).Return(usecase.SantaRemindResult{Sent: 2, Unreachable: 1}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/santa/rooms/"+roomID.String()+"/remind", nil)
	req.Header.Set("Authorization", "Bearer "+makeTestToken(user))

	status, body := doReq(t, newSantaApp(m), req)

	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"data":{"sent":2,"unreachable":1}}`, body)
}

func TestSantaRemind_RequiresLogin(t *testing.T) {
	m := &MockSantaUC{}
	status, _ := doReq(t, newSantaApp(m), httptest.NewRequest(http.MethodPost, "/api/v1/santa/rooms/"+uuid.NewString()+"/remind", nil))
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestSantaVerifyEmail_WrongCodeIs422(t *testing.T) {
	m := &MockSantaUC{}
	m.On("VerifyEmail", mock.Anything, "abcdefgh", mock.Anything, "000000").
		Return(usecase.SantaMe{}, fmt.Errorf("%w: неверный код", usecase.ErrSantaInvalid))

	status, body := doReq(t, newSantaApp(m), jsonReq("/api/v1/santa/r/abcdefgh/me/email/verify", `{"code":"000000"}`))

	assert.Equal(t, http.StatusUnprocessableEntity, status)
	assert.Contains(t, body, "неверный код")
}

func TestTelegramWebhook_RejectsMissingSecret(t *testing.T) {
	m := &MockSantaUC{}
	req := jsonReq("/api/v1/telegram/webhook", `{"message":{"chat":{"id":77},"text":"/start X"}}`)

	status, _ := doReq(t, newSantaApp(m), req)

	assert.Equal(t, http.StatusUnauthorized, status)
	m.AssertNotCalled(t, "TelegramStart", mock.Anything, mock.Anything, mock.Anything)
}

func TestSantaRequestEmailCode_IPLimit(t *testing.T) {
	m := &MockSantaUC{}
	m.On("RequestEmailCode", mock.Anything, "abcdefgh", mock.Anything, mock.Anything).Return(nil)
	app := newSantaApp(m)

	for i := 0; i < v1.EmailCodePerMinute; i++ {
		status, _ := doReq(t, app, jsonReq("/api/v1/santa/r/abcdefgh/me/email", `{"email":"a@example.com"}`))
		require.Equal(t, http.StatusOK, status, "запрос %d", i+1)
	}
	status, body := doReq(t, app, jsonReq("/api/v1/santa/r/abcdefgh/me/email", `{"email":"b@example.com"}`))

	assert.Equal(t, http.StatusTooManyRequests, status)
	assert.Contains(t, body, v1.EmailCodeLimitMessage)
	m.AssertNumberOfCalls(t, "RequestEmailCode", v1.EmailCodePerMinute)

	// Предел только на запрос кода: соседние маршруты участника работают.
	m.On("VerifyEmail", mock.Anything, "abcdefgh", mock.Anything, "123456").Return(usecase.SantaMe{}, nil)
	status, _ = doReq(t, app, jsonReq("/api/v1/santa/r/abcdefgh/me/email/verify", `{"code":"123456"}`))
	assert.Equal(t, http.StatusOK, status)
}

func TestSantaRequestEmailCode_PerAddressLimitIs429(t *testing.T) {
	m := &MockSantaUC{}
	m.On("RequestEmailCode", mock.Anything, "abcdefgh", mock.Anything, mock.Anything).Return(usecase.ErrSantaEmailLimit)

	status, body := doReq(t, newSantaApp(m), jsonReq("/api/v1/santa/r/abcdefgh/me/email", `{"email":"a@example.com"}`))

	assert.Equal(t, http.StatusTooManyRequests, status)
	assert.Contains(t, body, usecase.ErrSantaEmailLimit.Error())
}

// unavailableErr — как ошибка use case: свой текст, errors.Is — ErrSantaUnavailable.
type unavailableErr struct{ msg string }

func (e unavailableErr) Error() string { return e.msg }
func (e unavailableErr) Unwrap() error { return usecase.ErrSantaUnavailable }

func TestSantaChannelsUnavailableIs503(t *testing.T) {
	m := &MockSantaUC{}
	m.On("RequestEmailCode", mock.Anything, "abcdefgh", mock.Anything, mock.Anything).
		Return(unavailableErr{"отправка почты пока не настроена"})
	m.On("TelegramLink", mock.Anything, "abcdefgh", mock.Anything).
		Return("", unavailableErr{"подключение Telegram пока не настроено"})
	app := newSantaApp(m)

	status, body := doReq(t, app, jsonReq("/api/v1/santa/r/abcdefgh/me/email", `{"email":"a@example.com"}`))
	assert.Equal(t, http.StatusServiceUnavailable, status)
	assert.JSONEq(t, `{"error":"отправка почты пока не настроена"}`, body)

	status, body = doReq(t, app, httptest.NewRequest(http.MethodPost, "/api/v1/santa/r/abcdefgh/me/telegram", nil))
	assert.Equal(t, http.StatusServiceUnavailable, status)
	assert.Contains(t, body, "подключение Telegram пока не настроено")
}

func TestSantaChat_Get(t *testing.T) {
	m := &MockSantaUC{}
	m.On("GetChat", mock.Anything, "AbCd2345", usecase.SantaAuth{Token: "tok"}, usecase.SantaChatSanta).
		Return(usecase.SantaChat{With: usecase.SantaChatSanta}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/santa/r/AbCd2345/me/chat?with=santa", nil)
	req.Header.Set(v1.SantaTokenHeader, "tok")

	status, body := doReq(t, newSantaApp(m), req)

	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"data":{"with":"santa","messages":[]}}`, body, "пустой чат — массив, не null")
}

func TestSantaChat_Send(t *testing.T) {
	m := &MockSantaUC{}
	m.On("SendChat", mock.Anything, "AbCd2345", usecase.SantaAuth{Token: "tok"}, usecase.SantaChatReceiver, "Какой размер?").
		Return(usecase.SantaChatMessage{Mine: true, Body: "Какой размер?"}, nil)
	req := jsonReq("/api/v1/santa/r/AbCd2345/me/chat", `{"with":"receiver","body":"Какой размер?"}`)
	req.Header.Set(v1.SantaTokenHeader, "tok")

	status, body := doReq(t, newSantaApp(m), req)

	assert.Equal(t, http.StatusCreated, status)
	assert.Contains(t, body, `"mine":true`)
}

func TestSantaChat_Errors(t *testing.T) {
	for err, want := range map[error]int{
		usecase.ErrSantaChatLimit: http.StatusTooManyRequests,
		usecase.ErrSantaNotInDraw: http.StatusConflict,
		usecase.ErrSantaNotDrawn:  http.StatusConflict,
	} {
		m := &MockSantaUC{}
		m.On("SendChat", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(usecase.SantaChatMessage{}, err)
		status, body := doReq(t, newSantaApp(m), jsonReq("/api/v1/santa/r/AbCd2345/me/chat", `{"with":"santa","body":"x"}`))
		assert.Equal(t, want, status, err.Error())
		assert.Contains(t, body, err.Error())
	}
}

func TestTelegramWebhook_ReplyGoesToChat(t *testing.T) {
	m := &MockSantaUC{}
	m.On("TelegramReply", mock.Anything, int64(77), int64(500), "Спасибо!").Return(nil)
	req := jsonReq("/api/v1/telegram/webhook", `{"update_id":2,"message":{"message_id":501,"chat":{"id":77},"text":" Спасибо! ","reply_to_message":{"message_id":500}}}`)
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "hook-secret")

	status, _ := doReq(t, newSantaApp(m), req)

	assert.Equal(t, http.StatusOK, status)
	m.AssertExpectations(t)
}

func TestTelegramWebhook_ReplyWithoutTextIgnored(t *testing.T) {
	m := &MockSantaUC{}
	// Стикер или фото в ответ — текста нет.
	req := jsonReq("/api/v1/telegram/webhook", `{"message":{"chat":{"id":77},"reply_to_message":{"message_id":500}}}`)
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "hook-secret")

	status, _ := doReq(t, newSantaApp(m), req)

	assert.Equal(t, http.StatusOK, status)
	m.AssertNotCalled(t, "TelegramReply", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}
