package v1

import (
	"io"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"main/internal/controller/restapi/v1/response"
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

func (h *wishlistHandler) update(c *fiber.Ctx) error {
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

	wishlist, err := h.uc.Update(c.Context(), id, input)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(wishlist))
}

func (h *wishlistHandler) delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid wishlist ID"))
	}

	if err := h.uc.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(true))
}

func (h *wishlistHandler) parseWishlistInput(c *fiber.Ctx) (usecase.CreateWishlistInput, error) {
	input := usecase.CreateWishlistInput{
		Title:                c.FormValue("title"),
		Description:          c.FormValue("description"),
		ColorScheme:          c.FormValue("settings[colorScheme]"),
		ShowGiftAvailability: stringToBool(c.FormValue("settings[showGiftAvailability]")),
		LocationName:         c.FormValue("location[name]"),
		LocationLink:         c.FormValue("location[link]"),
	}

	if timeValue := c.FormValue("location[time]"); timeValue != "" {
		if t, err := time.Parse(time.RFC3339, timeValue); err == nil {
			input.LocationTime = t
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
