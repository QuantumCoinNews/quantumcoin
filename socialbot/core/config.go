package core

import (
	"log"
	"os"
)

type Config struct {
	AppEnv string

	// X
	XApiKey       string
	XApiSecret    string
	XAccessToken  string
	XAccessSecret string

	// Telegram
	TelegramBotToken string
	TelegramChatID   string

	// YouTube
	YouTubeApiKey string

	// General
	DataDir string
}

var AppConfig *Config

func LoadConfig() *Config {
	cfg := &Config{
		AppEnv: getEnv("APP_ENV", "dev"),

		XApiKey:       getEnv("X_API_KEY", ""),
		XApiSecret:    getEnv("X_API_SECRET", ""),
		XAccessToken:  getEnv("X_ACCESS_TOKEN", ""),
		XAccessSecret: getEnv("X_ACCESS_SECRET", ""),

		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:   getEnv("TELEGRAM_CHAT_ID", ""),

		YouTubeApiKey: getEnv("YOUTUBE_API_KEY", ""),

		DataDir: getEnv("DATA_DIR", "./data"),
	}

	AppConfig = cfg
	return cfg
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func MustLoadConfig() *Config {
	cfg := LoadConfig()

	if cfg.AppEnv == "" {
		log.Fatal("APP_ENV missing")
	}

	return cfg
}
