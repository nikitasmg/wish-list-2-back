package handler

import (
	"errors"
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
	// Получаем идентификатор пользователя
	err, userID := helpers.GetUserId(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve user ID"})
	}

	// Создаем переменную для хранения данных списка желаемого
	var wishlist model.CreateWishlist

	// Получаем материалы формы
	if err := c.BodyParser(&wishlist); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	// Заполняем поля структуры из вложенных данных
	if title := c.FormValue("title"); title != "" {
		wishlist.Title = title
	}

	if description := c.FormValue("description"); description != "" {
		wishlist.Description = description
	}

	// Получаем значения для настроек
	wishlist.Settings.ColorScheme = c.FormValue("settings[colorScheme]")
	wishlist.Settings.ShowGiftAvailability = helpers.StringToBool(c.FormValue("showGiftAvailability"))

	// Получаем значения для местоположения
	wishlist.Location.Name = c.FormValue("location[name]")
	wishlist.Location.Link = c.FormValue("location[link]")
	if timeValue := c.FormValue("location[time]"); timeValue != "" {
		if t, err := time.Parse(time.RFC3339, timeValue); err == nil {
			wishlist.Location.Time = t
		}
	}

	// Помещаем ваш файл в MinIO или другую файловую систему
	url, err := h.minioClient.CreateOneHandler(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to upload file"})
	}

	// Создаем новый список желаемого
	newWishlist := model.Wishlist{
		ID:          uuid.New(),
		UserID:      *userID,
		Title:       wishlist.Title,
		Description: wishlist.Description,
		Cover:       url,
		Settings: model.Settings{
			ColorScheme:          wishlist.Settings.ColorScheme,
			ShowGiftAvailability: wishlist.Settings.ShowGiftAvailability,
		},
		Location: model.Location{
			Name: wishlist.Location.Name,
			Time: wishlist.Location.Time,
			Link: wishlist.Location.Link,
		},
		PresentsCount: 0,
	}

	// Сохраняем новый список желаемого в базе данных
	if err = database.DB.Create(&newWishlist).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create Wishlist"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": newWishlist})
}

func (h *wishlistHandlers) GetOne(c *fiber.Ctx) error {
	id := c.Params("id")
	wishlistId, err := uuid.Parse(id) // Преобразуем строку в uuid.UUID
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid Wishlist ID"})
	}

	err, data := wishlistService.GetOne(wishlistId)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(200).JSON(fiber.Map{"data": data})
}

func (h *wishlistHandlers) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	wishlistId, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Не верный формат UUID"})
	}
	wishlist := model.Wishlist{ID: wishlistId}
	result := database.DB.Delete(&wishlist)
	if result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": result.Error.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": true})
}

func (h *wishlistHandlers) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	wishlistId, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Не верный формат UUID"})
	}

	// Пытаемся найти существующий список желаемого
	wishlist := model.Wishlist{ID: wishlistId}
	result := database.DB.First(&wishlist)
	if result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Вишлист с таким ID не существует"})
	}

	// Создаем новую структуру для обновления
	var updatedData model.CreateWishlist
	if err = c.BodyParser(&updatedData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input", "details": err.Error()})
	}

	// Теперь showGiftAvailability - это bool переменная

	// Обновляем все поля из запроса
	wishlist.Title = updatedData.Title
	wishlist.Description = updatedData.Description
	wishlist.Settings = model.Settings{
		ColorScheme:          c.FormValue("settings[colorScheme]"),
		ShowGiftAvailability: helpers.StringToBool(c.FormValue("settings[showGiftAvailability]")),
	}
	wishlist.Location = model.Location{
		Name: updatedData.Location.Name,
		Time: updatedData.Location.Time,
		Link: updatedData.Location.Link,
	}

	// Получаем файл, если он есть
	file, err := c.FormFile("file")
	if err != nil && !errors.Is(err, fiber.ErrBadRequest) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File processing error"})
	}

	if file != nil {
		// Если файл передан, загружаем его
		url, err := h.minioClient.CreateOneHandler(c)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to upload file"})
		}
		wishlist.Cover = url
	} else {
		wishlist.Cover = ""
	}

	// Сохраняем обновления в базе данных
	if err := database.DB.Save(&wishlist).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update Wishlist"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": wishlist})
}
