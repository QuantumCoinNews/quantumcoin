package platforms

import (
	"log"
	"os"
	"strings"

	"socialbot/content"
	"socialbot/core"
)

type YouTubePublisher struct {
	dryRun bool
	auth   *core.Auth
}

func NewYouTubePublisher(a *core.Auth) *YouTubePublisher {
	return &YouTubePublisher{
		dryRun: envBool("YT_DRY_RUN", true) || envBool("DRY_RUN", false),
		auth:   a,
	}
}

func (p *YouTubePublisher) Publish(item content.ContentItem) error {
	video := strings.TrimSpace(item.VideoPath)
	if video == "" {
		log.Printf("[youtube] skip: no video id=%s", sanitizeASCII(item.ID))
		return nil
	}

	title := strings.TrimSpace(item.Title)
	if title == "" {
		title = "Video"
	}

	privacy := strings.ToLower(strings.TrimSpace(os.Getenv("YT_PRIVACY")))
	if privacy == "" {
		privacy = "public"
	}
	if privacy != "public" && privacy != "unlisted" && privacy != "private" {
		privacy = "public"
	}

	if p.dryRun {
		log.Printf("[youtube][dry-run] file=%q title=%q privacy=%s", video, sanitizeASCII(title), privacy)
		return nil
	}

	// TODO: real YouTube upload (later)
	log.Printf("[youtube][mock-live] file=%q title=%q privacy=%s", video, sanitizeASCII(title), privacy)
	return nil
}
