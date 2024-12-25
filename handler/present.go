package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"main/database"
	"main/model"
	"time"
)

func CreatePresent(c *fiber.Ctx) error {
	var present model.Present
	var id = c.Params("id")
	if err := c.BodyParser(&present); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}
	wishListId, err := uuid.Parse(id) // Преобразуем строку в uuid.UUID
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid wishlist ID"})
	}
	// Создаем новый валидатор
	validate := validator.New()
	// Валидируем структуру
	if err = validate.Struct(present); err != nil {
		// Если есть ошибки валидации, извлекаем их и возвращаем клиенту
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	result := database.DB.First(model.WishList{ID: wishListId})

	if result.Error != nil {
		// Если есть ошибки валидации, извлекаем их и возвращаем клиенту
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Wishlist ID not found"})
	}

	// Создаем новый список желаемого и сохраняем его в базе данных
	newPresent := model.Present{
		ID:          uuid.New(),
		WishListID:  wishListId,
		Title:       present.Title,
		Description: present.Description,
		Cover:       present.Cover,
		Link:        present.Link,
		Reserved:    false,
		CreatedAt:   time.Now(),
	}
	// Сохраняем новый список желаемого в базе данных
	if err = database.DB.Create(&newPresent).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create wishlist"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": newPresent})
}
func GetPresents(c *fiber.Ctx) error {
	id := c.Params("id")
	wishListId, err := uuid.Parse(id) // Преобразуем строку в uuid.UUID
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid wishlist ID"})
	}

	var presents []model.Present
	result := database.DB.Where("wish_list_id = ?", wishListId).Find(&presents)

	if result.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Not found"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": presents})
}
