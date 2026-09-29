package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	App   AppConfig
	DB    DBConfig
	Auth  AuthConfig
	Minio MinioConfig
}

type AppConfig struct {
	Port           string
	CORSOrigin     string
	MinioPublicURL string
	Env            string
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

type MinioConfig struct {
	Endpoint     string
	BucketName   string
	RootUser     string
	RootPassword string
	UseSSL       bool
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		App: AppConfig{
			Port:           getEnv("PORT", "8080"),
			CORSOrigin:     getEnv("CORS_ORIGIN", "https://prosto-namekni.ru"),
			MinioPublicURL: getEnv("MINIO_PUBLIC_URL", "https://files.prosto-namekni.ru"),
			Env:            getEnv("APP_ENV", "production"),
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
		Minio: MinioConfig{
			Endpoint:     getEnv("MINIO_ENDPOINT", "minio:9000"),
			BucketName:   getEnv("MINIO_BUCKET_NAME", "wish-list-bucket"),
			RootUser:     getEnv("MINIO_ROOT_USER", "root"),
			RootPassword: getEnv("MINIO_ROOT_PASSWORD", "minio_password"),
			UseSSL:       getEnvAsBool("MINIO_USE_SSL", false),
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

func getEnvAsBool(key string, defaultValue bool) bool {
	if valueStr := getEnv(key, ""); valueStr != "" {
		if value, err := strconv.ParseBool(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}
