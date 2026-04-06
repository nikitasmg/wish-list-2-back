package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"main/internal/controller/restapi/v1/response"
	"main/internal/usecase"
)

type templateHandler struct {
	uc usecase.TemplateUseCase
}

func newTemplateHandler(uc usecase.TemplateUseCase) *templateHandler {
	return &templateHandler{uc: uc}
}

func (h *templateHandler) getPublic(c *fiber.Ctx) error {
	cursor := c.Query("cursor", "")
	templates, nextCursor, err := h.uc.GetPublic(c.Context(), 0, cursor)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
	}
	return c.JSON(fiber.Map{
		"data":       templates,
		"nextCursor": nextCursor,
	})
}

func (h *templateHandler) getMy(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}
	templates, err := h.uc.GetAllByUser(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(templates))
}

func (h *templateHandler) create(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}

	var body struct {
		WishlistID string `json:"wishlistId"`
		Name       string `json:"name"`
		IsPublic   bool   `json:"isPublic"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid input"))
	}
	wishlistID, err := uuid.Parse(body.WishlistID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid wishlistId"))
	}
	if body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("name is required"))
	}

	t, err := h.uc.Create(c.Context(), userID, usecase.CreateTemplateInput{
		WishlistID: wishlistID,
		Name:       body.Name,
		IsPublic:   body.IsPublic,
	})
	if err != nil {
		if err.Error() == "forbidden" {
			return c.Status(fiber.StatusForbidden).JSON(response.Error(err.Error()))
		}
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}
	return c.Status(fiber.StatusCreated).JSON(response.Data(t))
}

func (h *templateHandler) update(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid template ID"))
	}

	var body struct {
		Name     string `json:"name"`
		IsPublic bool   `json:"isPublic"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid input"))
	}

	t, err := h.uc.Update(c.Context(), id, userID, usecase.UpdateTemplateInput{
		Name:     body.Name,
		IsPublic: body.IsPublic,
	})
	if err != nil {
		if err.Error() == "forbidden" {
			return c.Status(fiber.StatusForbidden).JSON(response.Error(err.Error()))
		}
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(t))
}

func (h *templateHandler) delete(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid template ID"))
	}

	if err := h.uc.Delete(c.Context(), id, userID); err != nil {
		if err.Error() == "forbidden" {
			return c.Status(fiber.StatusForbidden).JSON(response.Error(err.Error()))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(true))
}

func (h *templateHandler) createWishlistFromTemplate(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}
	templateID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid template ID"))
	}

	var body struct {
		Title string `json:"title"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid input"))
	}
	if body.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("title is required"))
	}

	wishlist, err := h.uc.CreateWishlistFromTemplate(c.Context(), templateID, userID, body.Title)
	if err != nil {
		if err.Error() == "forbidden" {
			return c.Status(fiber.StatusForbidden).JSON(response.Error(err.Error()))
		}
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(err.Error()))
	}
	return c.Status(fiber.StatusCreated).JSON(response.Data(wishlist))
}
