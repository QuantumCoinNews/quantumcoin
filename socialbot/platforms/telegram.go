package platforms

import (
	"fmt"
	"log"
	"strings"
	"time"

	"socialbot/content"
	"socialbot/core"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramPublisher struct {
	bot    *tgbotapi.BotAPI
	chatID int64
	dryRun bool
}

func NewTelegramPublisher(a *core.Auth) *TelegramPublisher {
	if a == nil || a.TelegramBot == nil {
		return nil
	}
	return &TelegramPublisher{
		bot:    a.TelegramBot,
		chatID: a.TelegramChatID,
		dryRun: envBool("TG_DRY_RUN", false) || envBool("DRY_RUN", false),
	}
}

func (p *TelegramPublisher) Publish(item content.ContentItem) error {
	if p == nil || p.bot == nil {
		return fmt.Errorf("telegram: publisher not initialized")
	}

	caption := buildTelegramCaption(item)

	if p.dryRun {
		log.Printf("[telegram][dry-run] caption=%q video=%q image=%q album=%d",
			sanitizeASCII(caption),
			strings.TrimSpace(item.VideoPath),
			strings.TrimSpace(item.ImagePath),
			len(item.Album),
		)
		return nil
	}

	switch {
	case strings.TrimSpace(item.VideoPath) != "":
		msg := tgbotapi.NewVideo(p.chatID, tgbotapi.FilePath(strings.TrimSpace(item.VideoPath)))
		msg.Caption = caption
		_, err := p.bot.Send(msg)
		return err

	case len(item.Album) > 0:
		var media []interface{}
		for i, path := range item.Album {
			path = strings.TrimSpace(path)
			if path == "" {
				continue
			}
			m := tgbotapi.NewInputMediaPhoto(tgbotapi.FilePath(path))
			if i == 0 {
				m.Caption = caption
			}
			media = append(media, m)
		}
		if len(media) == 0 {
			return fmt.Errorf("telegram: album empty after filtering")
		}
		cfg := tgbotapi.NewMediaGroup(p.chatID, media)
		_, err := p.bot.SendMediaGroup(cfg)
		return err

	case strings.TrimSpace(item.ImagePath) != "":
		msg := tgbotapi.NewPhoto(p.chatID, tgbotapi.FilePath(strings.TrimSpace(item.ImagePath)))
		msg.Caption = caption
		_, err := p.bot.Send(msg)
		return err

	default:
		msg := tgbotapi.NewMessage(p.chatID, caption)
		_, err := p.bot.Send(msg)
		return err
	}
}

func buildTelegramCaption(it content.ContentItem) string {
	var b strings.Builder

	if s := strings.TrimSpace(it.Title); s != "" {
		b.WriteString(s)
		b.WriteString("\n")
	}
	if s := strings.TrimSpace(it.Caption); s != "" {
		b.WriteString(s)
		b.WriteString("\n")
	}
	if s := strings.TrimSpace(it.Text); s != "" {
		b.WriteString(s)
		b.WriteString("\n")
	}

	links := make([]string, 0, 3)
	if s := strings.TrimSpace(it.YTLink); s != "" {
		links = append(links, "YouTube: "+s)
	}
	if s := strings.TrimSpace(it.XLink); s != "" {
		links = append(links, "X: "+s)
	}
	if s := strings.TrimSpace(it.IGLink); s != "" {
		links = append(links, "Instagram: "+s)
	}
	if len(links) > 0 {
		b.WriteString(strings.Join(links, " | "))
		b.WriteString("\n")
	}

	if h := formatHashtags(it.Tags, 25); h != "" {
		b.WriteString("\n")
		b.WriteString(h)
		b.WriteString("\n")
	}

	b.WriteString("\n-- ")
	b.WriteString(time.Now().UTC().Format("2006-01-02 15:04"))
	b.WriteString(" UTC")

	return strings.TrimSpace(b.String())
}
