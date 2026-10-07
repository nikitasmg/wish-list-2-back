package v1

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"main/internal/controller/restapi/middleware"
	"main/internal/controller/restapi/v1/response"
	"main/internal/usecase"
)

// SantaTokenHeader — секрет участника без аккаунта. Фронт хранит его в
// localStorage и шлёт с каждым запросом к своей комнате.
const SantaTokenHeader = "X-Santa-Token"

// TelegramSecretHeader — Telegram кладёт сюда secret_token из setWebhook.
const TelegramSecretHeader = "X-Telegram-Bot-Api-Secret-Token"

type santaHandler struct {
	uc usecase.SantaUseCase
}

// NewSantaRouter вешает маршруты Тайного Санты. Вызывать ДО NewRouter: там
// защищённая группа с префиксом "" навешивает JWT на весь /api/v1, и
// публичные маршруты, зарегистрированные после неё, требовали бы вход.
func NewSantaRouter(router fiber.Router, jwtSecret, webhookSecret string, uc usecase.SantaUseCase) {
	h := &santaHandler{uc: uc}
	api := router.Group("/api/v1/santa")
	optional := middleware.JWTOptional(jwtSecret)

	api.Get("/r/:slug", h.invite)
	api.Post("/r/:slug/join", optional, h.join)
	api.Get("/r/:slug/me", optional, h.me)
	api.Patch("/r/:slug/me", optional, h.updateMe)
	api.Delete("/r/:slug/me", optional, h.leave)
	api.Post("/r/:slug/me/email", optional, h.requestEmailCode)
	api.Post("/r/:slug/me/email/verify", optional, h.verifyEmail)
	api.Post("/r/:slug/me/telegram", optional, h.telegramLink)

	rooms := api.Group("/rooms", middleware.JWTRequired401(jwtSecret))
	rooms.Get("", h.listRooms)
	rooms.Post("", h.createRoom)
	rooms.Get("/:id", h.getRoom)
	rooms.Patch("/:id", h.updateRoom)
	rooms.Delete("/:id", h.deleteRoom)
	rooms.Delete("/:id/participants/:pid", h.removeParticipant)
	rooms.Post("/:id/draw", h.draw)
	rooms.Post("/:id/redraw", h.redraw)
	rooms.Post("/:id/remind", h.remind)

	// Без секрета вебхук не включаем: иначе любой мог бы слать «апдейты».
	if webhookSecret != "" {
		router.Post("/api/v1/telegram/webhook", h.telegramWebhook(webhookSecret))
	}
}

func santaError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, usecase.ErrSantaNotFound):
		return c.Status(fiber.StatusNotFound).JSON(response.Error(usecase.ErrSantaNotFound.Error()))
	case errors.Is(err, usecase.ErrSantaDrawn):
		return c.Status(fiber.StatusConflict).JSON(response.Error(usecase.ErrSantaDrawn.Error()))
	case errors.Is(err, usecase.ErrSantaNotDrawn):
		return c.Status(fiber.StatusConflict).JSON(response.Error(usecase.ErrSantaNotDrawn.Error()))
	case errors.Is(err, usecase.ErrSantaAlreadyJoined):
		return c.Status(fiber.StatusConflict).JSON(response.Error(usecase.ErrSantaAlreadyJoined.Error()))
	case errors.Is(err, usecase.ErrSantaEmailTaken):
		return c.Status(fiber.StatusConflict).JSON(response.Error(usecase.ErrSantaEmailTaken.Error()))
	case errors.Is(err, usecase.ErrSantaTooSoon):
		return c.Status(fiber.StatusTooManyRequests).JSON(response.Error(usecase.ErrSantaTooSoon.Error()))
	case errors.Is(err, usecase.ErrSantaTooFew), errors.Is(err, usecase.ErrSantaInvalid):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error(err.Error()))
	}
	log.Printf("santa: %v", err)
	return c.Status(fiber.StatusInternalServerError).JSON(response.Error("внутренняя ошибка"))
}

type santaRoomBody struct {
	Title           string `json:"title"`
	Budget          *int   `json:"budget"`
	ExchangeDate    string `json:"exchangeDate"`
	DrawAt          string `json:"drawAt"`
	Message         string `json:"message"`
	OrganizerJoins  bool   `json:"organizerJoins"`
	OrganizerName   string `json:"organizerName"`
	OrganizerWishes string `json:"organizerWishes"`
}

func (b santaRoomBody) input() (usecase.SantaRoomInput, error) {
	in := usecase.SantaRoomInput{
		Title: b.Title, Budget: b.Budget, Message: b.Message, OrganizerJoins: b.OrganizerJoins,
		OrganizerName: b.OrganizerName, OrganizerWishes: b.OrganizerWishes,
	}
	if b.ExchangeDate != "" {
		d, err := time.Parse(time.DateOnly, b.ExchangeDate)
		if err != nil {
			return in, fmt.Errorf("%w: exchangeDate: неверный формат даты (нужен ГГГГ-ММ-ДД)", usecase.ErrSantaInvalid)
		}
		in.ExchangeDate = &d
	}
	if b.DrawAt != "" {
		t, err := time.Parse(time.RFC3339, b.DrawAt)
		if err != nil {
			return in, fmt.Errorf("%w: drawAt: неверный формат даты (нужен RFC 3339)", usecase.ErrSantaInvalid)
		}
		in.DrawAt = &t
	}
	return in, nil
}

type santaProfileBody struct {
	Name        string `json:"name"`
	Wishes      string `json:"wishes"`
	WishlistURL string `json:"wishlistUrl"`
}

func (b santaProfileBody) input() usecase.SantaProfileInput {
	return usecase.SantaProfileInput{Name: b.Name, Wishes: b.Wishes, WishlistURL: b.WishlistURL}
}

func santaAuth(c *fiber.Ctx) usecase.SantaAuth {
	return usecase.SantaAuth{Token: c.Get(SantaTokenHeader), UserID: getOptionalUserID(c)}
}

// ownerParams — пользователь из JWT и комната из пути. ok=false — ответ уже отправлен.
func ownerParams(c *fiber.Ctx) (userID, roomID uuid.UUID, ok bool, err error) {
	userID, err = getUserID(c)
	if err != nil {
		return uuid.Nil, uuid.Nil, false, c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}
	roomID, err = uuid.Parse(c.Params("id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, false, santaError(c, usecase.ErrSantaNotFound)
	}
	return userID, roomID, true, nil
}

func (h *santaHandler) createRoom(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}
	var body santaRoomBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid input"))
	}
	in, err := body.input()
	if err != nil {
		return santaError(c, err)
	}
	room, err := h.uc.CreateRoom(c.Context(), userID, in)
	if err != nil {
		return santaError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response.Data(room))
}

func (h *santaHandler) listRooms(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}
	rooms, err := h.uc.ListRooms(c.Context(), userID)
	if err != nil {
		return santaError(c, err)
	}
	if rooms == nil {
		rooms = []usecase.SantaRoomSummary{}
	}
	return c.JSON(response.Data(rooms))
}

func (h *santaHandler) getRoom(c *fiber.Ctx) error {
	userID, roomID, ok, err := ownerParams(c)
	if !ok {
		return err
	}
	details, err := h.uc.GetRoom(c.Context(), userID, roomID)
	if err != nil {
		return santaError(c, err)
	}
	return c.JSON(response.Data(details))
}

func (h *santaHandler) updateRoom(c *fiber.Ctx) error {
	userID, roomID, ok, err := ownerParams(c)
	if !ok {
		return err
	}
	var body santaRoomBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid input"))
	}
	in, err := body.input()
	if err != nil {
		return santaError(c, err)
	}
	room, err := h.uc.UpdateRoom(c.Context(), userID, roomID, in)
	if err != nil {
		return santaError(c, err)
	}
	return c.JSON(response.Data(room))
}

func (h *santaHandler) deleteRoom(c *fiber.Ctx) error {
	userID, roomID, ok, err := ownerParams(c)
	if !ok {
		return err
	}
	if err := h.uc.DeleteRoom(c.Context(), userID, roomID); err != nil {
		return santaError(c, err)
	}
	return c.JSON(response.Data(true))
}

func (h *santaHandler) removeParticipant(c *fiber.Ctx) error {
	userID, roomID, ok, err := ownerParams(c)
	if !ok {
		return err
	}
	pid, err := uuid.Parse(c.Params("pid"))
	if err != nil {
		return santaError(c, usecase.ErrSantaNotFound)
	}
	if err := h.uc.RemoveParticipant(c.Context(), userID, roomID, pid); err != nil {
		return santaError(c, err)
	}
	return c.JSON(response.Data(true))
}

func (h *santaHandler) draw(c *fiber.Ctx) error {
	userID, roomID, ok, err := ownerParams(c)
	if !ok {
		return err
	}
	if err := h.uc.Draw(c.Context(), userID, roomID); err != nil {
		return santaError(c, err)
	}
	return c.JSON(response.Data(true))
}

func (h *santaHandler) redraw(c *fiber.Ctx) error {
	userID, roomID, ok, err := ownerParams(c)
	if !ok {
		return err
	}
	if err := h.uc.Redraw(c.Context(), userID, roomID); err != nil {
		return santaError(c, err)
	}
	return c.JSON(response.Data(true))
}

func (h *santaHandler) invite(c *fiber.Ctx) error {
	inv, err := h.uc.GetInvite(c.Context(), c.Params("slug"))
	if err != nil {
		return santaError(c, err)
	}
	return c.JSON(response.Data(inv))
}

func (h *santaHandler) join(c *fiber.Ctx) error {
	var body santaProfileBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid input"))
	}
	res, err := h.uc.Join(c.Context(), c.Params("slug"), santaAuth(c), body.input())
	if err != nil {
		return santaError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response.Data(res))
}

func (h *santaHandler) me(c *fiber.Ctx) error {
	me, err := h.uc.GetMe(c.Context(), c.Params("slug"), santaAuth(c))
	if err != nil {
		return santaError(c, err)
	}
	return c.JSON(response.Data(me))
}

func (h *santaHandler) updateMe(c *fiber.Ctx) error {
	var body santaProfileBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid input"))
	}
	me, err := h.uc.UpdateMe(c.Context(), c.Params("slug"), santaAuth(c), body.input())
	if err != nil {
		return santaError(c, err)
	}
	return c.JSON(response.Data(me))
}

func (h *santaHandler) leave(c *fiber.Ctx) error {
	if err := h.uc.LeaveMe(c.Context(), c.Params("slug"), santaAuth(c)); err != nil {
		return santaError(c, err)
	}
	return c.JSON(response.Data(true))
}

func (h *santaHandler) requestEmailCode(c *fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid input"))
	}
	if err := h.uc.RequestEmailCode(c.Context(), c.Params("slug"), santaAuth(c), body.Email); err != nil {
		return santaError(c, err)
	}
	return c.JSON(response.Data(true))
}

func (h *santaHandler) verifyEmail(c *fiber.Ctx) error {
	var body struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid input"))
	}
	me, err := h.uc.VerifyEmail(c.Context(), c.Params("slug"), santaAuth(c), body.Code)
	if err != nil {
		return santaError(c, err)
	}
	return c.JSON(response.Data(me))
}

func (h *santaHandler) telegramLink(c *fiber.Ctx) error {
	url, err := h.uc.TelegramLink(c.Context(), c.Params("slug"), santaAuth(c))
	if err != nil {
		return santaError(c, err)
	}
	return c.JSON(response.Data(fiber.Map{"url": url}))
}

func (h *santaHandler) remind(c *fiber.Ctx) error {
	userID, roomID, ok, err := ownerParams(c)
	if !ok {
		return err
	}
	res, err := h.uc.Remind(c.Context(), userID, roomID)
	if err != nil {
		return santaError(c, err)
	}
	return c.JSON(response.Data(res))
}

type telegramUpdate struct {
	Message *struct {
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
}

// telegramWebhook всегда отвечает 200 на свои апдейты — иначе Telegram будет
// повторять их; ошибки только в лог.
func (h *santaHandler) telegramWebhook(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if subtle.ConstantTimeCompare([]byte(c.Get(TelegramSecretHeader)), []byte(secret)) != 1 {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		var u telegramUpdate
		if err := json.Unmarshal(c.Body(), &u); err != nil || u.Message == nil {
			return c.SendStatus(fiber.StatusOK)
		}
		text := strings.TrimSpace(u.Message.Text)
		if text == "/start" || strings.HasPrefix(text, "/start ") {
			token := strings.TrimSpace(strings.TrimPrefix(text, "/start"))
			if err := h.uc.TelegramStart(c.Context(), u.Message.Chat.ID, token); err != nil {
				log.Printf("santa: telegram start: %v", err)
			}
		}
		return c.SendStatus(fiber.StatusOK)
	}
}
