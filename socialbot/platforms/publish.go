package platforms

import (
	"fmt"
	"strings"

	"socialbot/content"
	"socialbot/core"
)

type Publishers struct {
	Telegram  *TelegramPublisher
	X         *XPublisher
	YouTube   *YouTubePublisher
	Instagram *InstagramPublisher
	TikTok    *TikTokPublisher
}

func NewPublishers(a *core.Auth) *Publishers {
	return &Publishers{
		Telegram:  NewTelegramPublisher(a),
		X:         NewXPublisher(a),
		YouTube:   NewYouTubePublisher(a),
		Instagram: NewInstagramPublisher(a),
		TikTok:    NewTikTokPublisher(a),
	}
}

func (p *Publishers) PublishAll(item content.ContentItem) error {
	var errs []string

	if p.X != nil {
		if err := p.X.Publish(item); err != nil {
			errs = append(errs, "x: "+err.Error())
		}
	}
	if p.Telegram != nil {
		if err := p.Telegram.Publish(item); err != nil {
			errs = append(errs, "telegram: "+err.Error())
		}
	}
	if p.YouTube != nil {
		if err := p.YouTube.Publish(item); err != nil {
			errs = append(errs, "youtube: "+err.Error())
		}
	}
	if p.Instagram != nil {
		if err := p.Instagram.Publish(item); err != nil {
			errs = append(errs, "instagram: "+err.Error())
		}
	}
	if p.TikTok != nil {
		if err := p.TikTok.Publish(item); err != nil {
			errs = append(errs, "tiktok: "+err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("publish errors: %s", strings.Join(errs, " | "))
	}
	return nil
}
