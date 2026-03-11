package core

import (
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Auth struct {
	TelegramBot    *tgbotapi.BotAPI
	TelegramChatID int64
}

func NewAuth() *Auth {
	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	chat := strings.TrimSpace(os.Getenv("TELEGRAM_CHAT_ID"))

	if token == "" || chat == "" {
		log.Fatal("Missing TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID in .env")
	}

	chatID, err := strconv.ParseInt(chat, 10, 64)
	if err != nil {
		log.Fatal("TELEGRAM_CHAT_ID must be int64 (example: -1001234567890)")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("Telegram bot init failed: %v", err)
	}

	return &Auth{
		TelegramBot:    bot,
		TelegramChatID: chatID,
	}
}
