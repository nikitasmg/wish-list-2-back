package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"main/database"
	"main/model"
	"main/pkg/minio"
	"main/services"
)

type PresentHandlers interface {
	GetOne(c *fiber.Ctx) error
	GetAll(c *fiber.Ctx) error
	Create(c *fiber.Ctx) error
	Update(c *fiber.Ctx) error
	Delete(c *fiber.Ctx) error
	Reserve(c *fiber.Ctx) error
	Release(c *fiber.Ctx) error
}

var presentService = services.NewPresentService()

type presentHandlers struct {
	minioClient minio.Client
}

func NewPresentService(minioClient minio.Client) PresentHandlers {
	return &presentHandlers{
		minioClient: minioClient,
	}
}

func (h *presentHandlers) GetOne(c *fiber.Ctx) error {
	id := c.Params("id")
	presentId, err := uuid.Parse(id) // Преобразуем строку в uuid.UUID
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid present ID"})
	}

	err, data := presentService.GetOne(presentId)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(200).JSON(fiber.Map{"data": data})
}

func (h *presentHandlers) Create(c *fiber.Ctx) error {
	var present model.CreatePresent
	var id = c.Params("wishlistId")
	if err := c.BodyParser(&present); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}
	wishlistId, err := uuid.Parse(id) // Преобразуем строку в uuid.UUID
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат UUID"})
	}
	// Создаем новый валидатор
	validate := validator.New()
	// Валидируем структуру
	if err = validate.Struct(present); err != nil {
		// Если есть ошибки валидации, извлекаем их и возвращаем клиенту
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	err, _ = wishlistService.GetOne(wishlistId)

	if err != nil {
		// Если есть ошибки валидации, извлекаем их и возвращаем клиенту
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Вишлист с таким ID не существует"})
	}
	err, _ = wishlistService.IncreasePresentsCount(wishlistId)
	if err != nil {
		// Если есть ошибки валидации, извлекаем их и возвращаем клиенту
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Ошибка при изменении количества подарков"})
	}
	url, err := h.minioClient.CreateOneHandler(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	// Создаем новый подарок желаемого и сохраняем его в базе данных
	newPresent := model.Present{
		ID:          uuid.New(),
		WishlistID:  wishlistId,
		Title:       present.Title,
		Description: present.Description,
		Cover:       url,
		Link:        present.Link,
		Reserved:    false,
	}
	// Сохраняем новый список желаемого в базе данных
	if err = database.DB.Create(&newPresent).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create Present"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": newPresent})
}

func (h *presentHandlers) GetAll(c *fiber.Ctx) error {
	id := c.Params("wishlistId")
	wishlistId, err := uuid.Parse(id) // Преобразуем строку в uuid.UUID
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат UUID"})
	}

	err, presents := presentService.GetAll(wishlistId)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(
		fiber.Map{"data": presents})
}

func (h *presentHandlers) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	presentId, err := uuid.Parse(id) // Преобразуем строку в uuid.UUID
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат UUID подарка"})
	}
	wishlistId := c.Params("wishlistId")
	wishlistParseId, err := uuid.Parse(wishlistId) // Преобразуем строку в uuid.UUID
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Неверный формат UUID вишлиста"})
	}

	err, _ = wishlistService.DecreasePresentsCount(wishlistParseId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Ошибка в изменении количества подарков"})
	}

	result := database.DB.Delete(&model.Present{ID: presentId})

	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Can't delete present"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": true})
}

func (h *presentHandlers) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	presentId, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Не верный формат UUID"})
	}

	// Создаем новую структуру для обновления
	var updatedData model.CreatePresent
	if err = c.BodyParser(&updatedData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input", "details": err.Error()})
	}

	// Пытаемся найти существующий подарок
	present := model.Present{ID: presentId}
	result := database.DB.First(&present)
	if result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Подарок с таким ID не существует"})
	}

	file, err := c.FormFile("file")

	// Обновляем все поля из запроса
	present.Title = updatedData.Title
	present.Description = updatedData.Description
	present.Link = updatedData.Link

	if file != nil {
		url, err := h.minioClient.CreateOneHandler(c)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		present.Cover = url
	} else {
		// Если файл не передан, можно удалить или оставить старое значение
		present.Cover = "" // или ничего не делать, в зависимости от вашей логики
	}

	// Сохраняем обновления в базе данных
	if err := database.DB.Save(&present).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update Present"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": present})
}

func (h *presentHandlers) Reserve(c *fiber.Ctx) error {
	id := c.Params("id")
	presentId, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid Present ID"})
	}

	err, present := presentService.GetOne(presentId)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Present not found"})
	}

	if present.Reserved {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Упс... Подарок уже был забронирован, пожалуйста перезагрузите страницу"})
	}

	present.Reserved = true

	// Сохраняем обновления в базе данных
	if err := database.DB.Save(&present).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update present"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": true})
}

func (h *presentHandlers) Release(c *fiber.Ctx) error {
	id := c.Params("id")
	presentId, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid Present ID"})
	}

	err, present := presentService.GetOne(presentId)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Present not found"})
	}

	present.Reserved = false

	// Сохраняем обновления в базе данных
	if err := database.DB.Save(&present).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update present"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": true})
}
