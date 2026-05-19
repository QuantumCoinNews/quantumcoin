package platforms

import (
	"fmt"
	"log"
	"strings"

	"socialbot/content"
	"socialbot/core"
)

type InstagramPublisher struct {
	dryRun bool
}

func NewInstagramPublisher(_ *core.Auth) *InstagramPublisher {
	return &InstagramPublisher{
		dryRun: envBool("IG_DRY_RUN", true) || envBool("DRY_RUN", false),
	}
}

func (p *InstagramPublisher) Publish(item content.ContentItem) error {
	kind, path := pickIGMedia(item)
	if kind == "" {
		return fmt.Errorf("instagram: no media (need videoPath or imagePath or album[0])")
	}

	caption := buildInstagramCaption(item)

	if p.dryRun {
		log.Printf("[instagram][dry-run] kind=%s file=%q caption=%q", kind, path, sanitizeASCII(caption))
		return nil
	}

	log.Printf("[instagram][mock-live] kind=%s file=%q caption=%q", kind, path, sanitizeASCII(caption))
	return nil
}

func pickIGMedia(it content.ContentItem) (kind, path string) {
	if strings.TrimSpace(it.VideoPath) != "" {
		return "reel", strings.TrimSpace(it.VideoPath)
	}
	if strings.TrimSpace(it.ImagePath) != "" {
		return "photo", strings.TrimSpace(it.ImagePath)
	}
	if len(it.Album) > 0 && strings.TrimSpace(it.Album[0]) != "" {
		return "carousel", strings.TrimSpace(it.Album[0])
	}
	return "", ""
}

func buildInstagramCaption(it content.ContentItem) string {
	var b strings.Builder

	if s := strings.TrimSpace(it.Title); s != "" {
		b.WriteString(s)
	}
	if s := strings.TrimSpace(it.Caption); s != "" {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(s)
	}
	if s := strings.TrimSpace(it.Text); s != "" {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(clipRunes(s, 900))
	}

	if h := formatHashtags(it.Tags, 20); h != "" {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(h)
	}

	return strings.TrimSpace(b.String())
}
