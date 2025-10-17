package main

import (
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Auth struct {
	TelegramBot  *tgbotapi.BotAPI
	TelegramChat string
}

func NewAuth() *Auth {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chat := os.Getenv("TELEGRAM_CHAT_ID")
	if token == "" || chat == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN veya TELEGRAM_CHAT_ID .env dosyasında tanımlı değil")
	}
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("Telegram bot init hata: %v", err)
	}
	return &Auth{
		TelegramBot:  bot,
		TelegramChat: chat,
	}
}
