package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds application configuration values.
type Config struct {
	AppPort           string
	DatabaseURL       string
	JWTSecret         string
	TokenExpires      time.Duration
	PaymeMerchantID   string
	PaymeMerchantKey  string
	TelegramBotToken  string
	TelegramAdminChat string
<<<<<<< HEAD
	PlumBaseURL       string
	PlumUsername      string
	PlumPassword      string
	PlumEnabled       bool
=======
>>>>>>> aa20ef04ed67ec5424fe0b2e816639ec249f073e
}

// Load reads environment variables and returns a populated Config.
func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		AppPort:           getEnv("APP_PORT", "8080"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/shafran?sslmode=disable"),
		JWTSecret:         getEnv("JWT_SECRET", "5f9a3c84a1d37b26e4e8725f9b8e22b987a81b7b19d47360f14b23c021e25f65b00b97b09cb8dc4abbd27fd9624b6df5"),
		TokenExpires:      getEnvDuration("JWT_TTL_HOURS", 24) * time.Hour,
		PaymeMerchantID:   getEnv("PAYME_MERCHANT_ID", ""),
		PaymeMerchantKey:  getEnv("PAYME_MERCHANT_KEY", ""),
		TelegramBotToken:  getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramAdminChat: getEnv("TELEGRAM_ADMIN_CHAT_ID", ""),
<<<<<<< HEAD
		PlumBaseURL:       getEnv("PLUM_BASE_URL", "https://pay.myuzcard.uz/api"),
		PlumUsername:      getEnv("PLUM_USERNAME", ""),
		PlumPassword:      getEnv("PLUM_PASSWORD", ""),
		PlumEnabled:       getEnv("PLUM_ENABLED", "false") == "true",
=======
>>>>>>> aa20ef04ed67ec5424fe0b2e816639ec249f073e
	}

	if cfg.AppPort == "" {
		log.Fatal("APP_PORT must be set")
	}

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET must be set")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvDuration(key string, fallback int) time.Duration {
	if value, ok := os.LookupEnv(key); ok {
		if parsed, err := strconv.Atoi(value); err == nil {
			return time.Duration(parsed)
		}
	}
	return time.Duration(fallback)
}
