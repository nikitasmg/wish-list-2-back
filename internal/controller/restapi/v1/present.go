package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"main/internal/controller/restapi/middleware"
	"main/internal/controller/restapi/v1/response"
	"main/internal/entity"
	"main/internal/usecase"
)

type presentHandler struct {
	uc usecase.PresentUseCase
}

func newPresentHandler(uc usecase.PresentUseCase) *presentHandler {
	return &presentHandler{uc: uc}
}

func (h *presentHandler) getOne(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid present ID"))
	}

	present, err := h.uc.GetByID(c.Context(), userID, id)
	if err != nil {
		return ownerError(c, err, fiber.StatusNotFound)
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

	// Кто забронировал — наружу не отдаём (поле помечено json:"-"), но текущему
	// гостю показываем его собственные брони, чтобы он мог их снять.
	guestID, _ := middleware.GuestIDFromCtx(c)
	for i := range presents {
		presents[i].ReservedByMe = presents[i].ReservedByGuest != "" &&
			presents[i].ReservedByGuest == guestID.String()
	}

	return c.JSON(response.Data(presents))
}

func (h *presentHandler) create(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}
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

	present, err := h.uc.Create(c.Context(), userID, wishlistID, input)
	if err != nil {
		return ownerError(c, err, fiber.StatusBadRequest)
	}
	return c.Status(fiber.StatusCreated).JSON(response.Data(present))
}

func (h *presentHandler) update(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid present ID"))
	}

	input, err := h.parsePresentInput(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}

	present, err := h.uc.Update(c.Context(), userID, id, input)
	if err != nil {
		return ownerError(c, err, fiber.StatusInternalServerError)
	}
	return c.JSON(response.Data(present))
}

func (h *presentHandler) delete(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid present ID"))
	}
	wishlistID, err := uuid.Parse(c.Params("wishlistId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid wishlist ID"))
	}

	if err := h.uc.Delete(c.Context(), userID, wishlistID, id); err != nil {
		return ownerError(c, err, fiber.StatusInternalServerError)
	}
	return c.JSON(response.Data(true))
}

func (h *presentHandler) reserve(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid present ID"))
	}

	guestID, ok := middleware.GuestIDFromCtx(c)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("не удалось определить гостя — включите куки и обновите страницу"))
	}

	if err := h.uc.Reserve(c.Context(), id, guestID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(true))
}

func (h *presentHandler) release(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid present ID"))
	}

	guestID, ok := middleware.GuestIDFromCtx(c)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("не удалось определить гостя — включите куки и обновите страницу"))
	}

	if err := h.uc.Release(c.Context(), id, guestID); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(response.Error(err.Error()))
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
		Links:       parseLinks(c),
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

// parseLinks собирает ссылки на магазины из формы.
//
// Принимаем оба вида: links[0], links[1]… от нового фронта и одиночное поле
// link от старого — иначе форма подарка перестала бы сохранять ссылку в тот
// момент, когда бэк уже обновлён, а фронт ещё нет.
func parseLinks(c *fiber.Ctx) []string {
	var links []string

	for i := 0; i < entity.MaxPresentLinks; i++ {
		value := c.FormValue(fmt.Sprintf("links[%d]", i))
		if value == "" {
			continue
		}
		links = append(links, value)
	}

	if len(links) == 0 {
		if single := c.FormValue("link"); single != "" {
			links = append(links, single)
		}
	}

	return links
}
