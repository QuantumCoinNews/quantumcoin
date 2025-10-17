package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramPublisher struct {
	bot    *tgbotapi.BotAPI
	chatID string // @channel username veya numeric chat id
}

func NewTelegramPublisher(a *Auth) *TelegramPublisher {
	return &TelegramPublisher{
		bot:    a.TelegramBot,
		chatID: a.TelegramChat,
	}
}

func (p *TelegramPublisher) Publish(item ContentItem) error {
	caption := buildTgCaption(item)

	switch {
	case item.VideoPath != "":
		video := tgbotapi.NewVideo(p.chatID, tgbotapi.FilePath(item.VideoPath))
		video.Caption = caption
		_, err := p.bot.Send(video)
		return err

	case len(item.Album) > 0:
		var media []interface{}
		for i, path := range item.Album {
			m := tgbotapi.NewInputMediaPhoto(tgbotapi.FilePath(path))
			if i == 0 {
				m.Caption = caption
			}
			media = append(media, m)
		}
		cfg := tgbotapi.MediaGroupConfig{
			ChannelUsername: p.chatID,
			Media:           media,
		}
		_, err := p.bot.SendMediaGroup(cfg)
		return err

	case item.ImagePath != "":
		photo := tgbotapi.NewPhoto(p.chatID, tgbotapi.FilePath(item.ImagePath))
		photo.Caption = caption
		_, err := p.bot.Send(photo)
		return err

	default:
		msg := tgbotapi.MessageConfig{
			BaseChat: tgbotapi.BaseChat{
				ChannelUsername: p.chatID,
			},
			Text: caption,
		}
		_, err := p.bot.Send(msg)
		return err
	}
}

func buildTgCaption(it ContentItem) string {
	var b strings.Builder

	if it.Title != "" {
		b.WriteString(it.Title)
		b.WriteString("\n")
	}
	if it.Caption != "" {
		b.WriteString(it.Caption)
		b.WriteString("\n")
	}
	if it.Text != "" {
		b.WriteString(it.Text)
		b.WriteString("\n")
	}

	links := make([]string, 0, 3)
	if it.YTLink != "" {
		links = append(links, "YouTube: "+it.YTLink)
	}
	if it.XLink != "" {
		links = append(links, "X: "+it.XLink)
	}
	if it.IGLink != "" {
		links = append(links, "Instagram: "+it.IGLink)
	}
	if len(links) > 0 {
		b.WriteString(strings.Join(links, " | "))
		b.WriteString("\n")
	}

	if len(it.Tags) > 0 {
		b.WriteString("\n")
		for _, t := range it.Tags {
			if t == "" {
				continue
			}
			if !strings.HasPrefix(t, "#") {
				b.WriteString("#")
			}
			b.WriteString(t)
			b.WriteString(" ")
		}
	}

	b.WriteString(fmt.Sprintf("\n\n— %s UTC", time.Now().UTC().Format("2006-01-02 15:04")))
	return b.String()
}

// (opsiyonel) basit logging yardımcıları
func tgLogErr(prefix string, err error) {
	if err != nil {
		log.Printf("[telegram] %s: %v", prefix, err)
	}
}
