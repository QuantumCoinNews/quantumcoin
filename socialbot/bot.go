package main

import (
	"log"
	"sync"
	"time"

	"socialbot/content"
	"socialbot/core"
	"socialbot/platforms"
)

type SocialBot struct {
	auth       *core.Auth
	publishers *platforms.Publishers
	mu         sync.Mutex
	running    bool
}

func NewSocialBot(a *core.Auth) *SocialBot {
	return &SocialBot{
		auth:       a,
		publishers: platforms.NewPublishers(a),
	}
}

func (b *SocialBot) DoAutoShare() {
	b.mu.Lock()
	if b.running {
		log.Printf("DoAutoShare skipped: already running")
		b.mu.Unlock()
		return
	}
	b.running = true
	b.mu.Unlock()

	start := time.Now()
	defer func() {
		b.mu.Lock()
		b.running = false
		b.mu.Unlock()
		log.Printf("DoAutoShare done (%.1fs)", time.Since(start).Seconds())
	}()

	item, err := content.PickToday()
	if err != nil {
		log.Printf("PickToday failed: %v", err)
		return
	}
	log.Printf("Selected content ID=%s Title=%q", item.ID, item.Title)

	if err := b.publishers.PublishAll(item); err != nil {
		log.Printf("PublishAll error: %v", err)
	}
}
