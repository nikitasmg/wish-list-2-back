package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"main/database"
	"main/model"
	"os"
	"time"
)

var jwtSecret = os.Getenv("JWT_SECRET")

func setToken(user model.User) (string, error) {
	database.DB.First(&user)
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
		// Если есть ошибки валидации, извлекаем их и возвращаем клиенту
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	// Хеширование пароля
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	result := database.DB.Create(&model.User{ID: uuid.New(), Username: user.Username, Password: string(hashedPassword)})
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not register user"})
	}

	tokenString, err := setToken(user)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create token"})
	}
	return c.JSON(fiber.Map{"token": tokenString})
}

func Login(c *fiber.Ctx) error {
	var user model.User
	if err := c.BodyParser(&user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	var currentUser model.User
	result := database.DB.Where("username = ?", user.Username).First(&currentUser)

	if result.Error != nil || bcrypt.CompareHashAndPassword([]byte(currentUser.Password), []byte(user.Password)) != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials"})
	}

	tokenString, err := setToken(user)

	if err != nil {
		log.WithError(err).Error("JWT token signing")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create token"})
	}

	return c.JSON(fiber.Map{"token": tokenString})
}

func Me(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*jwt.Token) // Получаем токен
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Invalid token"})
	}
	claims := user.Claims.(jwt.MapClaims)
	return c.Status(200).JSON(fiber.Map{"user": claims})
}
