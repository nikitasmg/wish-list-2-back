package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"main/database"
	"main/helpers"
	"main/model"
	"main/pkg/minio"
	"main/services"
	"time"
)

type WishlistHandlers interface {
	GetAll(c *fiber.Ctx) error
	GetOne(c *fiber.Ctx) error
	Create(c *fiber.Ctx) error
	Update(c *fiber.Ctx) error
	Delete(c *fiber.Ctx) error
}

var wishlistService = services.NewWishlistService()

type wishlistHandlers struct {
	minioClient minio.Client
}

func NewWishlistHandlers(minioClient minio.Client) WishlistHandlers {
	return &wishlistHandlers{
		minioClient: minioClient,
	}
}

func (h *wishlistHandlers) GetAll(c *fiber.Ctx) error {
	err, userID := helpers.GetUserId(c)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	err, wishlists := wishlistService.GetAll(userID)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": wishlists})
}

func (h *wishlistHandlers) Create(c *fiber.Ctx) error {
	err, userID := helpers.GetUserId(c)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	var Wishlist model.CreateWishlist
	if err = c.BodyParser(&Wishlist); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}
	// Создаем новый валидатор
	validate := validator.New()
	// Валидируем структуру
	if err = validate.Struct(Wishlist); err != nil {
		// Если есть ошибки валидации, извлекаем их и возвращаем клиенту
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	url, err := h.minioClient.CreateOneHandler(c)
	// Создаем новый список желаемого и сохраняем его в базе данных
	newWishlist := model.Wishlist{
		ID:          uuid.New(),
		UserID:      *userID,
		Title:       Wishlist.Title,
		Description: Wishlist.Description,
		Cover:       url,
		ColorScheme: Wishlist.ColorScheme,
		CreatedAt:   time.Now(),
	}
	// Сохраняем новый список желаемого в базе данных
	if err = database.DB.Create(&newWishlist).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create Wishlist"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": newWishlist})
}

func (h *wishlistHandlers) GetOne(c *fiber.Ctx) error {
	id := c.Params("id")
	WishlistId, err := uuid.Parse(id) // Преобразуем строку в uuid.UUID
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid Wishlist ID"})
	}

	err, data := wishlistService.GetOne(WishlistId)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(200).JSON(fiber.Map{"data": data})
}

func (h *wishlistHandlers) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	WishlistId, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Не верный формат UUID"})
	}
	Wishlist := model.Wishlist{ID: WishlistId}
	result := database.DB.Delete(&Wishlist)
	if result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": result.Error.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": true})
}

func (h *wishlistHandlers) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	WishlistId, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Не верный формат UUID"})
	}

	// Пытаемся найти существующий список желаемого
	Wishlist := model.Wishlist{ID: WishlistId}
	result := database.DB.First(&Wishlist)

	if result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Вишлист с таким ID не сущуесвтует"})
	}

	// Создаем новую структуру для обновления
	updateData := model.CreateWishlist{}
	if err = c.BodyParser(&updateData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	// Обновляем только те поля, которые были предоставлены
	if updateData.Title != "" {
		Wishlist.Title = updateData.Title
	}
	if updateData.Description != "" {
		Wishlist.Description = updateData.Description
	}
	if updateData.File != nil {
		url, err := h.minioClient.CreateOneHandler(c)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		Wishlist.Cover = url
	}
	if updateData.ColorScheme != "" {
		Wishlist.ColorScheme = updateData.ColorScheme
	}

	Wishlist.UpdatedAt = time.Now()

	// Сохраняем обновления в базе данных
	if err := database.DB.Save(&Wishlist).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update Wishlist"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": Wishlist})
}
