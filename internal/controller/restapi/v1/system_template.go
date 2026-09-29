package v1

import (
	"github.com/gofiber/fiber/v2"

	"main/internal/controller/restapi/v1/response"
	"main/internal/entity"
	"main/internal/usecase/systemtemplate"
)

type systemTemplateHandler struct{}

func newSystemTemplateHandler() *systemTemplateHandler { return &systemTemplateHandler{} }

// getAll — список шаблонов вместе с категориями для фильтров.
func (h *systemTemplateHandler) getAll(c *fiber.Ctx) error {
	templates, err := systemtemplate.All()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
	}

	return c.JSON(response.Data(fiber.Map{
		"templates":  templates,
		"categories": entity.SystemTemplateCategories,
	}))
}

func (h *systemTemplateHandler) getOne(c *fiber.Ctx) error {
	tpl, err := systemtemplate.Get(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(response.Error(err.Error()))
	}
	return c.JSON(response.Data(tpl))
}
