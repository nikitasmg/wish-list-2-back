package handler

import (
	"fmt"
	tgverifier "github.com/electrofocus/telegram-auth-verifier"
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
//func verifyTelegramAuth(botToken string, authData map[string]string, receivedHash string) error {
//	// 1. Извлекаем hash и удаляем его из данных
//	delete(authData, "hash")
//
//	// 2. Сортируем ключи в алфавитном порядке
//	keys := make([]string, 0, len(authData))
//	for k := range authData {
//		keys = append(keys, k)
//	}
//	sort.Strings(keys)
//
//	// 3. Формируем data-check-string
//	var dataCheckArr []string
//	for _, key := range keys {
//		dataCheckArr = append(dataCheckArr, fmt.Sprintf("%s=%s", key, authData[key]))
//	}
//	dataCheckString := strings.Join(dataCheckArr, "\n")
//
//	sha256hash := sha256.New()
//	io.WriteString(sha256hash, botToken)
//	hmachash := hmac.New(sha256.New, sha256hash.Sum(nil))
//	io.WriteString(hmachash, dataCheckString)
//	ss := hex.EncodeToString(hmachash.Sum(nil))
//	if receivedHash != ss {
//		return errors.New("Invalid signature")
//	}
//
//	// Отладочный вывод
//	log.Printf("DataCheckString:\n%s", dataCheckString)
//	log.Printf("DataCheckString (hex): %x", []byte(dataCheckString))
//
//	// 4. Вычисляем секретный ключ (SHA256 от токена бота)
//	secretKey := sha256.Sum256([]byte(botToken))
//	log.Printf("SecretKey (hex): %x", secretKey[:])
//
//	// 5. Вычисляем HMAC-SHA256
//	h := hmac.New(sha256.New, secretKey[:])
//	h.Write([]byte(dataCheckString))
//	expectedHash := hex.EncodeToString(h.Sum(nil))
//	expectedHash = strings.ToLower(expectedHash) // Telegram использует lowercase
//
//	// 6. Сравниваем хэши
//	receivedHash = strings.ToLower(receivedHash)
//	if expectedHash != receivedHash {
//		return fmt.Errorf("invalid hash (expected: %s, received: %s)", expectedHash, receivedHash)
//	}
//
//	// 7. Проверяем auth_date
//	authDate, ok := authData["auth_date"]
//	if !ok {
//		return fmt.Errorf("auth_date is missing")
//	}
//	authTimestamp, err := strconv.ParseInt(authDate, 10, 64)
//	if err != nil {
//		return fmt.Errorf("invalid auth_date: %v", err)
//	}
//	if time.Since(time.Unix(authTimestamp, 0)) > 24*time.Hour {
//		return fmt.Errorf("auth data is too old")
//	}
//
//	return nil
//}

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

	// Преобразование ID из строки в int64
	userID, err := strconv.ParseInt(data.ID, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID format",
		})
	}

	// Преобразование AuthDate из строки в int64
	authDate, err := strconv.ParseInt(data.AuthDate, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid auth date format",
		})
	}

	creds := tgverifier.Credentials{
		ID:        userID,
		FirstName: data.FirstName,
		Username:  data.Username,
		AuthDate:  authDate,
		Hash:      data.Hash,
	}

	// Проверяем подлинность
	if err := creds.Verify([]byte(botToken)); err != nil {
		fmt.Println("Auth failed:", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Authentication failed"})
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
