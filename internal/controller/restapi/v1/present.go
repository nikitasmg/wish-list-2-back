package v1

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"main/internal/controller/restapi/v1/response"
	"main/internal/usecase"
)

type presentHandler struct {
	uc usecase.PresentUseCase
}

func newPresentHandler(uc usecase.PresentUseCase) *presentHandler {
	return &presentHandler{uc: uc}
}

func (h *presentHandler) getOne(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid present ID"))
	}

	present, err := h.uc.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(present))
}

func (h *presentHandler) getAll(c *fiber.Ctx) error {
	wishlistID, err := uuid.Parse(c.Params("wishlistId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid wishlist ID"))
	}

	presents, err := h.uc.GetAllByWishlist(c.Context(), wishlistID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(presents))
}

func (h *presentHandler) create(c *fiber.Ctx) error {
	wishlistID, err := uuid.Parse(c.Params("wishlistId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid wishlist ID"))
	}

	input, err := h.parsePresentInput(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}
	if input.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("title is required"))
	}

	present, err := h.uc.Create(c.Context(), wishlistID, input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}
	return c.Status(fiber.StatusCreated).JSON(response.Data(present))
}

func (h *presentHandler) update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid present ID"))
	}

	input, err := h.parsePresentInput(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}

	present, err := h.uc.Update(c.Context(), id, input)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(present))
}

func (h *presentHandler) delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid present ID"))
	}
	wishlistID, err := uuid.Parse(c.Params("wishlistId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid wishlist ID"))
	}

	if err := h.uc.Delete(c.Context(), wishlistID, id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(true))
}

func (h *presentHandler) reserve(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid present ID"))
	}

	if err := h.uc.Reserve(c.Context(), id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(true))
}

func (h *presentHandler) release(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid present ID"))
	}

	if err := h.uc.Release(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(true))
}

func (h *presentHandler) join(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid present ID"))
	}
	if err := h.uc.Join(c.Context(), id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(true))
}

func (h *presentHandler) leave(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid present ID"))
	}
	if err := h.uc.Leave(c.Context(), id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(true))
}

var validSources = map[string]bool{
	"ozon": true, "wildberries": true, "yamarket": true, "other": true,
}

func validatePresentMeta(source, originalURL string) error {
	if source == "" {
		return nil
	}
	if !validSources[source] {
		return errors.New("invalid source: must be ozon, wildberries, yamarket, or other")
	}
	if originalURL == "" {
		return errors.New("original_url is required when source is set")
	}
	return nil
}

func (h *presentHandler) parsePresentInput(c *fiber.Ctx) (usecase.CreatePresentInput, error) {
	input := usecase.CreatePresentInput{
		Title:       c.FormValue("title"),
		Description: c.FormValue("description"),
		Link:        c.FormValue("link"),
		PriceStr:    c.FormValue("price"),
		CoverURL:    c.FormValue("cover_url"),
		Type:        c.FormValue("type"),
	}

	source := c.FormValue("source")
	originalURL := c.FormValue("original_url")
	if err := validatePresentMeta(source, originalURL); err != nil {
		return input, err
	}
	input.Source = source
	input.OriginalURL = originalURL
	input.Category = c.FormValue("category")
	input.Brand = c.FormValue("brand")

	if raw := c.FormValue("images"); raw != "" {
		var imgs []string
		if err := json.Unmarshal([]byte(raw), &imgs); err != nil {
			return input, errors.New("invalid images: must be JSON array")
		}
		input.Images = imgs
	}
	if raw := c.FormValue("links"); raw != "" {
		var links []string
		if err := json.Unmarshal([]byte(raw), &links); err != nil {
			return input, errors.New("invalid links: must be JSON array")
		}
		input.Links = links
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

		if len(data) > usecase.MaxFileSize {
			return input, errors.New("файл слишком большой: максимум 10MB")
		}

		input.CoverData = data
		input.CoverName = file.Filename
	}

	return input, nil
}
