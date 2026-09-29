package v1

import (
	"strconv"

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
	page, _ := strconv.Atoi(c.Query("page", "1"))
	userID := getOptionalUserID(c)

	templates, hasMore, err := h.uc.GetPublic(c.Context(), 0, page, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
	}
	return c.JSON(fiber.Map{
		"data":    templates,
		"hasMore": hasMore,
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

func (h *templateHandler) like(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}
	templateID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid template ID"))
	}

	result, err := h.uc.Like(c.Context(), userID, templateID)
	if err != nil {
		switch err.Error() {
		case "already liked":
			return c.Status(fiber.StatusConflict).JSON(response.Error(err.Error()))
		case "forbidden":
			return c.Status(fiber.StatusForbidden).JSON(response.Error(err.Error()))
		case "template not found":
			return c.Status(fiber.StatusNotFound).JSON(response.Error(err.Error()))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(result))
}

func (h *templateHandler) unlike(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}
	templateID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid template ID"))
	}

	result, err := h.uc.Unlike(c.Context(), userID, templateID)
	if err != nil {
		switch err.Error() {
		case "not liked", "template not found":
			return c.Status(fiber.StatusNotFound).JSON(response.Error(err.Error()))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(result))
}
