package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
	"strconv"
	"strings"
	"time"
)

var jwtSecret = os.Getenv("JWT_SECRET")

type TelegramAuthData struct {
	ID        string `json:"id"`         // ID пользователя
	FirstName string `json:"first_name"` // Имя пользователя
	Username  string `json:"username"`   // Имя пользователя (username)
	AuthDate  string `json:"auth_date"`  // Время авторизации в формате RFC3339
	Hash      string `json:"hash"`       // Хэш для проверки подлинности
}

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

// verifyTelegramAuth проверяет подпись данных от Telegram.
func verifyTelegramAuth(botToken string, data TelegramAuthData, hash string) error {
	// 1. Создаем data-check-string в строго заданном порядке
	dataCheckStrings := []string{
		fmt.Sprintf("auth_date=%s", data.AuthDate),
		fmt.Sprintf("first_name=%s", data.FirstName),
		fmt.Sprintf("id=%s", data.ID),
		fmt.Sprintf("username=%s", data.Username),
	}

	dataCheckString := strings.Join(dataCheckStrings, "\n")
	log.Printf("DataCheckString:\n%s", dataCheckString) // Убрал лишний \n в конце

	// 2. Вычисляем secret_key как SHA256 от токена бота
	secretKey := sha256.Sum256([]byte(botToken))

	// 3. Вычисляем HMAC-SHA256 от data-check-string
	h := hmac.New(sha256.New, secretKey[:])
	h.Write([]byte(dataCheckString))
	expectedHash := hex.EncodeToString(h.Sum(nil))

	// Исправлено: выводим expectedHash, а не botToken
	log.Printf("expectedHash: %s", expectedHash)
	log.Printf("received hash: %s", hash)

	// 4. Сравниваем хэши
	if expectedHash != hash {
		return fmt.Errorf("invalid hash")
	}

	// 5. Проверяем auth_date
	authTimestamp, err := strconv.ParseInt(data.AuthDate, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid auth_date: %v", err)
	}
	authTime := time.Unix(authTimestamp, 0)
	if time.Since(authTime) > 24*time.Hour {
		return fmt.Errorf("auth data expired")
	}

	return nil
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

	return c.JSON(fiber.Map{"token": tokenString})
}

func Authenticate(c *fiber.Ctx) error {
	var data TelegramAuthData

	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "BOT_TOKEN is not set"})
	}

	err := verifyTelegramAuth(botToken, data, data.Hash)
	if err != nil {
		log.Println("Ошибка в проверке хэша")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication failed"})
	}

	newUser := model.User{
		ID:       uuid.New(),
		Username: data.Username,
		Password: data.ID,
	}

	var currentUser model.User
	result := database.DB.Where("username = ?", newUser.Username).First(&currentUser)

	if result.Error != nil {
		database.DB.Create(&newUser)
	}

	tokenString, err := setToken(newUser)

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

func Logout(c *fiber.Ctx) error {
	helpers.ClearCookies(c, "token")
	return c.Status(200).JSON(fiber.Map{"message": "Logout successful"})
}
