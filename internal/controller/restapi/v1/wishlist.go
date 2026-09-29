package v1

import (
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"main/internal/controller/restapi/middleware"
	"main/internal/controller/restapi/v1/response"
	"main/internal/entity"
	"main/internal/usecase"
)

type wishlistHandler struct {
	uc usecase.WishlistUseCase
}

func newWishlistHandler(uc usecase.WishlistUseCase) *wishlistHandler {
	return &wishlistHandler{uc: uc}
}

func (h *wishlistHandler) getAll(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}

	wishlists, err := h.uc.GetAllByUser(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(wishlists))
}

func (h *wishlistHandler) getOne(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid wishlist ID"))
	}

	wishlist, err := h.uc.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(wishlist))
}

func (h *wishlistHandler) getByShortID(c *fiber.Ctx) error {
	shortID := c.Params("shortId")
	if shortID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("shortId is required"))
	}

	guestID, _ := middleware.GuestIDFromCtx(c)

	wishlist, err := h.uc.GetByShortID(c.Context(), shortID, guestID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(wishlist))
}

func (h *wishlistHandler) create(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}

	input, err := h.parseWishlistInput(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}
	if input.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("title is required"))
	}

	wishlist, err := h.uc.Create(c.Context(), userID, input)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
	}
	return c.Status(fiber.StatusCreated).JSON(response.Data(wishlist))
}

func (h *wishlistHandler) createConstructor(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}

	input, err := h.parseConstructorInput(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}
	if input.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("title is required"))
	}

	wishlist, err := h.uc.CreateConstructor(c.Context(), userID, input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}
	return c.Status(fiber.StatusCreated).JSON(response.Data(wishlist))
}

func (h *wishlistHandler) createFromTemplate(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}

	var body struct {
		TemplateID string `json:"template_id"`
		Title      string `json:"title"`
		EventDate  string `json:"event_date"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid JSON"))
	}
	if body.TemplateID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("template_id is required"))
	}

	input := usecase.CreateFromTemplateInput{
		TemplateID: body.TemplateID,
		Title:      body.Title,
	}
	if body.EventDate != "" {
		t, err := time.Parse(time.RFC3339, body.EventDate)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(response.Error("event_date должен быть датой в формате RFC3339"))
		}
		input.EventDate = &t
	}

	wishlist, err := h.uc.CreateFromTemplate(c.Context(), userID, input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}
	return c.Status(fiber.StatusCreated).JSON(response.Data(wishlist))
}

func (h *wishlistHandler) updateBlocks(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid wishlist ID"))
	}

	var blocks []entity.Block
	if err := c.BodyParser(&blocks); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid blocks JSON"))
	}

	expectedUpdatedAt, err := ifMatchVersion(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}

	wishlist, err := h.uc.UpdateBlocks(c.Context(), userID, id, blocks, expectedUpdatedAt)
	if errors.Is(err, usecase.ErrVersionConflict) {
		return conflictResponse(c, err, wishlist)
	}
	if err != nil {
		return ownerError(c, err, fiber.StatusBadRequest)
	}
	return c.JSON(response.Data(wishlist))
}

// ifMatchVersion — версия вишлиста, которую держит клиент.
//
// Заголовок, а не поле в теле: так запрос без заголовка остаётся рабочим —
// проверка версии в этом случае просто не включается, и старый клиент не
// ломается на ровном месте.
func ifMatchVersion(c *fiber.Ctx) (time.Time, error) {
	header := c.Get(fiber.HeaderIfMatch)
	if header == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, header)
	if err != nil {
		return time.Time{}, errors.New("If-Match должен быть временем в формате RFC3339")
	}
	return parsed, nil
}

// conflictResponse отдаёт 409 вместе с актуальным вишлистом: фронту есть что
// показать и с чем слить правку.
func conflictResponse(c *fiber.Ctx, err error, current entity.Wishlist) error {
	return c.Status(fiber.StatusConflict).JSON(fiber.Map{
		"error": err.Error(),
		"data":  current,
	})
}

func (h *wishlistHandler) update(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid wishlist ID"))
	}

	input, err := h.parseWishlistInput(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}
	if input.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("поле название обязательно"))
	}

	expectedUpdatedAt, err := ifMatchVersion(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}

	wishlist, err := h.uc.Update(c.Context(), userID, id, input, expectedUpdatedAt)
	if errors.Is(err, usecase.ErrVersionConflict) {
		return conflictResponse(c, err, wishlist)
	}
	if err != nil {
		return ownerError(c, err, fiber.StatusInternalServerError)
	}
	return c.JSON(response.Data(wishlist))
}

func (h *wishlistHandler) delete(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid wishlist ID"))
	}

	if err := h.uc.Delete(c.Context(), userID, id); err != nil {
		return ownerError(c, err, fiber.StatusInternalServerError)
	}
	return c.JSON(response.Data(true))
}

func (h *wishlistHandler) parseWishlistInput(c *fiber.Ctx) (usecase.CreateWishlistInput, error) {
	input := usecase.CreateWishlistInput{
		Title:                c.FormValue("title"),
		Description:          c.FormValue("description"),
		CoverURL:             c.FormValue("cover_url"),
		ColorScheme:          c.FormValue("settings[colorScheme]"),
		ShowGiftAvailability: stringToBool(c.FormValue("settings[showGiftAvailability]")),
		PresentsLayout:       c.FormValue("settings[presentsLayout]"),
		LocationName:         c.FormValue("location[name]"),
		LocationLink:         c.FormValue("location[link]"),
		Occasion:             c.FormValue("occasion"),
	}

	if timeValue := c.FormValue("location[time]"); timeValue != "" {
		if t, err := time.Parse(time.RFC3339, timeValue); err == nil {
			input.LocationTime = t
		}
	}

	if eventDate := c.FormValue("eventDate"); eventDate != "" {
		t, err := time.Parse(time.RFC3339, eventDate)
		if err != nil {
			return input, errors.New("eventDate должен быть датой в формате RFC3339")
		}
		input.EventDate = &t
	}

	if base := c.FormValue("settings[customScheme][base]"); base != "" {
		input.CustomScheme = &entity.CustomScheme{
			Base:   base,
			Accent: c.FormValue("settings[customScheme][accent]"),
		}
	}

	file, err := c.FormFile("file")
	if err == nil && file != nil {
		f, err := file.Open()
		if err != nil {
			return input, err
		}
		defer f.Close()
		data, err := io.ReadAll(f)
		if err != nil {
			return input, err
		}
		input.CoverData = data
		input.CoverName = file.Filename
	}

	return input, nil
}

func (h *wishlistHandler) parseConstructorInput(c *fiber.Ctx) (usecase.CreateConstructorInput, error) {
	var body struct {
		Title                string               `json:"title"`
		Description          string               `json:"description"`
		CoverURL             string               `json:"cover_url"`
		ColorScheme          string               `json:"color_scheme"`
		ShowGiftAvailability bool                 `json:"show_gift_availability"`
		PresentsLayout       string               `json:"presents_layout"`
		LocationName         string               `json:"location_name"`
		LocationLink         string               `json:"location_link"`
		LocationTime         string               `json:"location_time"`
		EventDate            string               `json:"event_date"`
		Occasion             string               `json:"occasion"`
		CustomScheme         *entity.CustomScheme `json:"custom_scheme"`
		Blocks               []entity.Block       `json:"blocks"`
	}

	if err := c.BodyParser(&body); err != nil {
		return usecase.CreateConstructorInput{}, err
	}

	input := usecase.CreateConstructorInput{
		Title:                body.Title,
		Description:          body.Description,
		CoverURL:             body.CoverURL,
		ColorScheme:          body.ColorScheme,
		ShowGiftAvailability: body.ShowGiftAvailability,
		PresentsLayout:       body.PresentsLayout,
		LocationName:         body.LocationName,
		LocationLink:         body.LocationLink,
		Occasion:             body.Occasion,
		CustomScheme:         body.CustomScheme,
		Blocks:               body.Blocks,
	}

	if body.LocationTime != "" {
		if t, err := time.Parse(time.RFC3339, body.LocationTime); err == nil {
			input.LocationTime = t
		}
	}

	if body.EventDate != "" {
		t, err := time.Parse(time.RFC3339, body.EventDate)
		if err != nil {
			return usecase.CreateConstructorInput{}, errors.New("event_date должен быть датой в формате RFC3339")
		}
		input.EventDate = &t
	}

	// Ensure block Data fields are valid JSON
	for i := range input.Blocks {
		if input.Blocks[i].Data == nil {
			input.Blocks[i].Data = json.RawMessage("{}")
		}
	}

	return input, nil
}
