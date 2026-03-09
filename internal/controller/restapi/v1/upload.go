package v1

import (
	"io"

	"github.com/gofiber/fiber/v2"

	"main/internal/controller/restapi/v1/response"
	"main/internal/usecase"
)

type uploadHandler struct {
	uc usecase.UploadUseCase
}

func newUploadHandler(uc usecase.UploadUseCase) *uploadHandler {
	return &uploadHandler{uc: uc}
}

// upload — загрузка одного файла
func (h *uploadHandler) upload(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("file is required"))
	}

	f, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error("failed to open file"))
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error("failed to read file"))
	}

	result, err := h.uc.Upload(c.Context(), file.Filename, data)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
	}

	return c.JSON(fiber.Map{"url": result.URL})
}

// bulkUpload — загрузка нескольких файлов
// Ожидает multipart с полями files[0], files[1], ...
func (h *uploadHandler) bulkUpload(c *fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid multipart form"))
	}

	fileHeaders, ok := form.File["files"]
	if !ok || len(fileHeaders) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("files are required"))
	}

	var inputs []usecase.FileInput
	for i, fh := range fileHeaders {
		f, err := fh.Open()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(response.Error("failed to open file"))
		}
		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(response.Error("failed to read file"))
		}
		inputs = append(inputs, usecase.FileInput{Index: i, Name: fh.Filename, Data: data})
	}

	results, err := h.uc.BulkUpload(c.Context(), inputs)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
	}

	return c.JSON(response.Data(results))
}
