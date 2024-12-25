package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"main/database"
	"main/helpers"
	"main/model"
	"time"
)

func GetAllWishlists(c *fiber.Ctx) error {
	err, userID := helpers.GetUserId(c)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	var wishLists []model.WishList
	result := database.DB.Where("user_id = ?", userID).Find(&wishLists)

	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": result.Error.Error()})
	}
	return c.Status(200).JSON(fiber.Map{"data": wishLists})
}

func CreateWishlist(c *fiber.Ctx) error {
	err, userID := helpers.GetUserId(c)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	var wishList model.WishList
	if err = c.BodyParser(&wishList); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}
	// Создаем новый валидатор
	validate := validator.New()
	// Валидируем структуру
	if err = validate.Struct(wishList); err != nil {
		// Если есть ошибки валидации, извлекаем их и возвращаем клиенту
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Создаем новый список желаемого и сохраняем его в базе данных
	newWishList := model.WishList{
		ID:          uuid.New(),
		UserID:      *userID,
		Title:       wishList.Title,
		Description: wishList.Description,
		Cover:       wishList.Cover,
		CreatedAt:   time.Now(),
	}
	// Сохраняем новый список желаемого в базе данных
	if err = database.DB.Create(&newWishList).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create wishlist"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": newWishList})
}

func GetWishList(c *fiber.Ctx) error {
	id := c.Params("id")
	wishListId, err := uuid.Parse(id) // Преобразуем строку в uuid.UUID
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid wishlist ID"})
	}

	wishList := model.WishList{ID: wishListId}
	result := database.DB.First(wishList)

	if result.Error != nil {
		return c.Status(404).JSON(fiber.Map{"error": result.Error.Error()})
	}

	return c.Status(200).JSON(fiber.Map{"data": wishList})
}
