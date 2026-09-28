package v1

import (
	"github.com/gofiber/fiber/v2"

	"main/internal/controller/restapi/v1/response"
	"main/internal/entity"
	"main/internal/usecase/template"
)

type templateHandler struct{}

func newTemplateHandler() *templateHandler { return &templateHandler{} }

// getAll — список шаблонов вместе с категориями для фильтров.
func (h *templateHandler) getAll(c *fiber.Ctx) error {
	templates, err := template.All()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
	}

	return c.JSON(response.Data(fiber.Map{
		"templates":  templates,
		"categories": entity.TemplateCategories,
	}))
}

func (h *templateHandler) getOne(c *fiber.Ctx) error {
	tpl, err := template.ByID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(tpl))
}
