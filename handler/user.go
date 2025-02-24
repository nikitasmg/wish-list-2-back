package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"main/database"
	"main/helpers"
	"main/model"
	"os"
	"time"
)

var jwtSecret = os.Getenv("JWT_SECRET")

func setToken(user model.User) (string, error) {
	claims := &model.Claims{
		Username: user.Username,
		Id:       user.ID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 24 * 30).Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(jwtSecret))

	return tokenString, err
}

func Register(c *fiber.Ctx) error {
	var user model.User
	if err := c.BodyParser(&user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	// Создаем новый валидатор
	validate := validator.New()

	// Валидируем структуру
	if err := validate.Struct(user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	var currentUser model.User
	err := database.DB.Where("username = ?", user.Username).First(&currentUser).Error
	if err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Пользователь с таким username уже существует"})
	}

	// Хеширование пароля
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	newUser := model.User{
		ID:       uuid.New(),
		Username: user.Username,
		Password: string(hashedPassword),
	}
	result := database.DB.Create(&newUser)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Не удалось зарегистрировать пользователя"})
	}

	tokenString, err := setToken(newUser)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create token"})
	}

	// Установка токена в куку
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    tokenString,
		Expires:  time.Now().Add(time.Hour * 24 * 30), // Установите желаемое время жизни куки
		Secure:   true,                                // Используйте true, если сайт работает по HTTPS
		HTTPOnly: true,
		SameSite: fiber.CookieSameSiteLaxMode,
		Path:     "/",
		Domain:   "get-my-wishlist.ru",
	})

	return c.JSON(fiber.Map{"message": "User registered successfully"})
}

func Login(c *fiber.Ctx) error {
	var user model.User
	if err := c.BodyParser(&user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	var currentUser model.User
	result := database.DB.Where("username = ?", user.Username).First(&currentUser)

	if result.Error != nil || bcrypt.CompareHashAndPassword([]byte(currentUser.Password), []byte(user.Password)) != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Неверный логин или пароль"})
	}

	tokenString, err := setToken(currentUser)

	if err != nil {
		log.WithError(err).Error("JWT token signing")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create token"})
	}

	// Установка токена в куку
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    tokenString,
		Expires:  time.Now().Add(time.Hour * 24 * 30), // Установите желаемое время жизни куки
		Secure:   true,                                // Используйте true, если сайт работает по HTTPS
		HTTPOnly: true,
		SameSite: fiber.CookieSameSiteLaxMode,
		Path:     "/",
		Domain:   "get-my-wishlist.ru",
	})

	return c.JSON(fiber.Map{"message": "Login successful"})
}

func Me(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*jwt.Token) // Получаем токен
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Invalid token"})
	}
	claims := user.Claims.(jwt.MapClaims)
	return c.Status(200).JSON(fiber.Map{"user": claims})
}

func Logout(c *fiber.Ctx) error {
	helpers.ClearCookies(c, "token")
	return c.Status(200).JSON(fiber.Map{"message": "Logout successful"})
}
