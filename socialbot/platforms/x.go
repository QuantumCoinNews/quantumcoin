package platforms

import (
	"log"
	"strings"
	"time"

	"socialbot/content"
	"socialbot/core"
)

type XPublisher struct {
	dryRun bool
}

func NewXPublisher(_ *core.Auth) *XPublisher {
	return &XPublisher{
		dryRun: envBool("X_DRY_RUN", true) || envBool("DRY_RUN", false),
	}
}

func (p *XPublisher) Publish(item content.ContentItem) error {
	text := buildXText(item)
	media := pickXMedia(item)

	if p.dryRun {
		if media != "" {
			log.Printf("[x][dry-run] post=%q media=%q", sanitizeASCII(text), media)
		} else {
			log.Printf("[x][dry-run] post=%q", sanitizeASCII(text))
		}
		return nil
	}

	log.Printf("[x][mock-live] post=%q media=%q", sanitizeASCII(text), media)
	return nil
}

func buildXText(it content.ContentItem) string {
	var parts []string

	if s := strings.TrimSpace(it.Title); s != "" {
		parts = append(parts, s)
	}
	if s := strings.TrimSpace(it.Caption); s != "" {
		parts = append(parts, s)
	}
	if s := strings.TrimSpace(it.Text); s != "" {
		parts = append(parts, clipRunes(s, 220))
	}

	if s := strings.TrimSpace(it.YTLink); s != "" {
		parts = append(parts, s)
	} else if s := strings.TrimSpace(it.XLink); s != "" {
		parts = append(parts, s)
	} else if s := strings.TrimSpace(it.IGLink); s != "" {
		parts = append(parts, s)
	}

	if h := formatHashtags(it.Tags, 6); h != "" {
		parts = append(parts, h)
	}

	if envBool("X_ADD_TIMESTAMP", false) {
		parts = append(parts, time.Now().UTC().Format("2006-01-02 15:04")+" UTC")
	}

	out := strings.Join(parts, "\n\n")
	return clipRunes(out, 260)
}

func pickXMedia(it content.ContentItem) string {
	if s := strings.TrimSpace(it.VideoPath); s != "" {
		return s
	}
	if s := strings.TrimSpace(it.ImagePath); s != "" {
		return s
	}
	if len(it.Album) > 0 && strings.TrimSpace(it.Album[0]) != "" {
		return strings.TrimSpace(it.Album[0])
	}
	return ""
}
