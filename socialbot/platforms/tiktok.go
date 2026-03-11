package platforms

import (
	"fmt"
	"log"
	"strings"

	"socialbot/content"
	"socialbot/core"
)

type TikTokPublisher struct {
	dryRun bool
}

func NewTikTokPublisher(_ *core.Auth) *TikTokPublisher {
	return &TikTokPublisher{
		dryRun: envBool("TT_DRY_RUN", true) || envBool("DRY_RUN", false),
	}
}

func (p *TikTokPublisher) Publish(item content.ContentItem) error {
	video := strings.TrimSpace(item.VideoPath)
	if video == "" {
		return fmt.Errorf("tiktok: videoPath is required")
	}

	caption := buildTikTokCaption(item)

	if p.dryRun {
		log.Printf("[tiktok][dry-run] file=%q caption=%q", video, sanitizeASCII(caption))
		return nil
	}

	log.Printf("[tiktok][mock-live] file=%q caption=%q", video, sanitizeASCII(caption))
	return nil
}

func buildTikTokCaption(it content.ContentItem) string {
	var b strings.Builder

	if s := strings.TrimSpace(it.Title); s != "" {
		b.WriteString(s)
	}
	if s := strings.TrimSpace(it.Caption); s != "" {
		if b.Len() > 0 {
			b.WriteString(" - ")
		}
		b.WriteString(s)
	}

	if h := formatHashtags(it.Tags, 15); h != "" {
		b.WriteString("\n")
		b.WriteString(h)
	}

	return strings.TrimSpace(b.String())
}
