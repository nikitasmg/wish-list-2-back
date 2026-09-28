package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"main/internal/controller/restapi/middleware"
	"main/internal/controller/restapi/v1/response"
	"main/internal/usecase"
)

type guestDataHandler struct {
	uc usecase.GuestDataUseCase
}

func newGuestDataHandler(uc usecase.GuestDataUseCase) *guestDataHandler {
	return &guestDataHandler{uc: uc}
}

// blockScope — общий разбор пути /wishlists/:wishlistId/blocks/:blockId/...
func (h *guestDataHandler) blockScope(c *fiber.Ctx) (uuid.UUID, string, error) {
	wishlistID, err := uuid.Parse(c.Params("wishlistId"))
	if err != nil {
		return uuid.Nil, "", c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid wishlist ID"))
	}
	blockID := c.Params("blockId")
	if blockID == "" {
		return uuid.Nil, "", c.Status(fiber.StatusBadRequest).JSON(response.Error("blockId is required"))
	}
	return wishlistID, blockID, nil
}

// RSVP

func (h *guestDataHandler) submitRSVP(c *fiber.Ctx) error {
	wishlistID, blockID, errResp := h.blockScope(c)
	if errResp != nil {
		return errResp
	}

	var body struct {
		Name     string `json:"name"`
		Going    bool   `json:"going"`
		PlusOne  int    `json:"plusOne"`
		Kids     int    `json:"kids"`
		Menu     string `json:"menu"`
		Transfer bool   `json:"transfer"`
		Comment  string `json:"comment"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid JSON"))
	}

	guestID, _ := middleware.GuestIDFromCtx(c)
	result, err := h.uc.SubmitRSVP(c.Context(), wishlistID, blockID, guestID, usecase.RSVPInput{
		Name:     body.Name,
		Going:    body.Going,
		PlusOne:  body.PlusOne,
		Kids:     body.Kids,
		Menu:     body.Menu,
		Transfer: body.Transfer,
		Comment:  body.Comment,
	})
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(result))
}

func (h *guestDataHandler) myRSVP(c *fiber.Ctx) error {
	_, blockID, errResp := h.blockScope(c)
	if errResp != nil {
		return errResp
	}

	guestID, _ := middleware.GuestIDFromCtx(c)
	result, err := h.uc.MyRSVP(c.Context(), blockID, guestID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(result))
}

func (h *guestDataHandler) rsvpSummary(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}
	wishlistID, blockID, errResp := h.blockScope(c)
	if errResp != nil {
		return errResp
	}

	summary, err := h.uc.OwnerRSVPSummary(c.Context(), userID, wishlistID, blockID)
	if err != nil {
		return ownerError(c, err, fiber.StatusInternalServerError)
	}
	return c.JSON(response.Data(summary))
}

// Голосование

func (h *guestDataHandler) pollResults(c *fiber.Ctx) error {
	_, blockID, errResp := h.blockScope(c)
	if errResp != nil {
		return errResp
	}

	guestID, _ := middleware.GuestIDFromCtx(c)
	results, err := h.uc.PollResults(c.Context(), blockID, guestID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(results))
}

func (h *guestDataHandler) vote(c *fiber.Ctx) error {
	wishlistID, blockID, errResp := h.blockScope(c)
	if errResp != nil {
		return errResp
	}

	var body struct {
		Option int `json:"option"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid JSON"))
	}

	guestID, _ := middleware.GuestIDFromCtx(c)
	results, err := h.uc.Vote(c.Context(), wishlistID, blockID, guestID, body.Option)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(results))
}

// Плейлист

func (h *guestDataHandler) tracks(c *fiber.Ctx) error {
	_, blockID, errResp := h.blockScope(c)
	if errResp != nil {
		return errResp
	}

	guestID, _ := middleware.GuestIDFromCtx(c)
	tracks, err := h.uc.Tracks(c.Context(), blockID, guestID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(tracks))
}

func (h *guestDataHandler) suggestTrack(c *fiber.Ctx) error {
	wishlistID, blockID, errResp := h.blockScope(c)
	if errResp != nil {
		return errResp
	}

	var body struct {
		Title string `json:"title"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid JSON"))
	}

	guestID, _ := middleware.GuestIDFromCtx(c)
	tracks, err := h.uc.SuggestTrack(c.Context(), wishlistID, blockID, guestID, body.Title)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}
	return c.Status(fiber.StatusCreated).JSON(response.Data(tracks))
}

func (h *guestDataHandler) toggleTrackVote(c *fiber.Ctx) error {
	trackID, err := uuid.Parse(c.Params("trackId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid track ID"))
	}

	guestID, _ := middleware.GuestIDFromCtx(c)
	tracks, err := h.uc.ToggleTrackVote(c.Context(), trackID, guestID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(tracks))
}

// Гостевая книга

func (h *guestDataHandler) guestbook(c *fiber.Ctx) error {
	_, blockID, errResp := h.blockScope(c)
	if errResp != nil {
		return errResp
	}

	guestID, _ := middleware.GuestIDFromCtx(c)
	entries, err := h.uc.Guestbook(c.Context(), blockID, guestID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(entries))
}

func (h *guestDataHandler) addGuestbookEntry(c *fiber.Ctx) error {
	wishlistID, blockID, errResp := h.blockScope(c)
	if errResp != nil {
		return errResp
	}

	var body struct {
		Name     string `json:"name"`
		Text     string `json:"text"`
		PhotoURL string `json:"photoUrl"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid JSON"))
	}

	guestID, _ := middleware.GuestIDFromCtx(c)
	entry, err := h.uc.AddGuestbookEntry(c.Context(), wishlistID, blockID, guestID, usecase.GuestbookInput{
		Name: body.Name, Text: body.Text, PhotoURL: body.PhotoURL,
	})
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}
	return c.Status(fiber.StatusCreated).JSON(response.Data(entry))
}

func (h *guestDataHandler) ownerGuestbook(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}
	wishlistID, blockID, errResp := h.blockScope(c)
	if errResp != nil {
		return errResp
	}

	entries, err := h.uc.OwnerGuestbook(c.Context(), userID, wishlistID, blockID)
	if err != nil {
		return ownerError(c, err, fiber.StatusInternalServerError)
	}
	return c.JSON(response.Data(entries))
}

func (h *guestDataHandler) setGuestbookHidden(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}
	entryID, err := uuid.Parse(c.Params("entryId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid entry ID"))
	}

	var body struct {
		Hidden bool `json:"hidden"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid JSON"))
	}

	if err := h.uc.OwnerSetGuestbookHidden(c.Context(), userID, entryID, body.Hidden); err != nil {
		return ownerError(c, err, fiber.StatusInternalServerError)
	}
	return c.JSON(response.Data(true))
}
