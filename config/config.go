package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	App    AppConfig
	DB     DBConfig
	Auth   AuthConfig
	S3     S3Config
	Notify NotifyConfig
}

// NotifyConfig — уведомления Тайного Санты.
type NotifyConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	MailFrom     string
	// TelegramBotToken — SANTA_BOT_TOKEN, по умолчанию бот входа BOT_TOKEN.
	TelegramBotToken      string
	TelegramBotUsername   string
	TelegramWebhookSecret string
	// SantaPublicURL — адрес поддомена для ссылок в письмах и сообщениях.
	SantaPublicURL string
}

type AppConfig struct {
	Port       string
	CORSOrigin string
	Env        string
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type AuthConfig struct {
	JWTSecret    string
	BotToken     string
	CookieDomain string
}

type S3Config struct {
	Endpoint         string
	Region           string
	BucketName       string
	AccessKeyID      string
	SecretAccessKey  string
	UseSSL           bool
	PublicURL        string
	AutoCreateBucket bool
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		App: AppConfig{
			Port:       getEnv("PORT", "8080"),
			CORSOrigin: getEnv("CORS_ORIGIN", "https://prosto-namekni.ru"),
			Env:        getEnv("APP_ENV", "production"),
		},
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "postgres"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "wishlist_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Auth: AuthConfig{
			JWTSecret:    getEnv("JWT_SECRET", ""),
			BotToken:     getEnv("BOT_TOKEN", ""),
			CookieDomain: getEnv("COOKIE_DOMAIN", "prosto-namekni.ru"),
		},
		S3: loadS3Config(),
		Notify: NotifyConfig{
			SMTPHost:              getEnv("SMTP_HOST", ""),
			SMTPPort:              getEnvAsInt("SMTP_PORT", 587),
			SMTPUser:              getEnv("SMTP_USER", ""),
			SMTPPassword:          getEnv("SMTP_PASSWORD", ""),
			MailFrom:              getEnv("MAIL_FROM", "Тайный Санта <santa@prosto-namekni.ru>"),
			TelegramBotToken:      getEnvNonEmpty("SANTA_BOT_TOKEN", getEnv("BOT_TOKEN", "")),
			TelegramBotUsername:   getEnv("BOT_USERNAME", ""),
			TelegramWebhookSecret: getEnv("TELEGRAM_WEBHOOK_SECRET", ""),
			SantaPublicURL:        getEnv("SANTA_PUBLIC_URL", "https://santa.prosto-namekni.ru"),
		},
	}

	if cfg.Auth.JWTSecret == "" {
		log.Println("WARNING: JWT_SECRET is not set")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if valueStr := getEnv(key, ""); valueStr != "" {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if valueStr := getEnv(key, ""); valueStr != "" {
		if value, err := strconv.ParseBool(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}

// getEnvNonEmpty — как getEnv, но пустое значение тоже считается незаданным
// (env_file и .env.example оставляют переменные пустыми).
func getEnvNonEmpty(key, defaultValue string) string {
	if value := getEnv(key, ""); value != "" {
		return value
	}
	return defaultValue
}

// S3_* settings take precedence; MINIO_* remains supported for local development.
func loadS3Config() S3Config {
	bucket := getEnvNonEmpty("S3_BUCKET_NAME", getEnvNonEmpty("MINIO_BUCKET_NAME", "wish-list-bucket"))
	publicURL := getEnvNonEmpty("S3_PUBLIC_URL", "")
	if publicURL == "" && getEnvNonEmpty("S3_ENDPOINT", "") == "" {
		publicURL = strings.TrimRight(getEnvNonEmpty("MINIO_PUBLIC_URL", "https://files.prosto-namekni.ru"), "/") + "/" + bucket
	}
	useSSL := getEnvAsBool("MINIO_USE_SSL", false)
	if getEnvNonEmpty("S3_ENDPOINT", "") != "" {
		useSSL = true
	}
	return S3Config{
		Endpoint:         getEnvNonEmpty("S3_ENDPOINT", getEnvNonEmpty("MINIO_ENDPOINT", "minio-internal:9000")),
		Region:           getEnvNonEmpty("S3_REGION", ""),
		BucketName:       bucket,
		AccessKeyID:      getEnvNonEmpty("S3_ACCESS_KEY_ID", getEnvNonEmpty("MINIO_ROOT_USER", "")),
		SecretAccessKey:  getEnvNonEmpty("S3_SECRET_ACCESS_KEY", getEnvNonEmpty("MINIO_ROOT_PASSWORD", "")),
		UseSSL:           getEnvAsBool("S3_USE_SSL", useSSL),
		PublicURL:        publicURL,
		AutoCreateBucket: getEnvAsBool("S3_AUTO_CREATE_BUCKET", false),
	}
}
