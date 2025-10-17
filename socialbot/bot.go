package main

import (
	"log"
	"sync"
	"time"
)

type SocialBot struct {
	auth       *Auth
	publishers *Publishers
	mu         sync.Mutex
	running    bool
}

func NewSocialBot(a *Auth) *SocialBot {
	return &SocialBot{
		auth:       a,
		publishers: NewPublishers(a),
	}
}

// DoAutoShare -> günlük otomatik paylaşım akışı
func (b *SocialBot) DoAutoShare() {
	b.mu.Lock()
	if b.running {
		log.Printf("DoAutoShare atlandı: zaten çalışıyor")
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
		log.Printf("DoAutoShare bitti (%.1fs)", time.Since(start).Seconds())
	}()

	item, err := PickToday()
	if err != nil {
		log.Printf("İçerik seçilemedi: %v", err)
		return
	}
	log.Printf("Gönderim ID=%s Title=%q", item.ID, item.Title)

	if err := b.publishers.PublishAll(item); err != nil {
		log.Printf("Publish hatası: %v", err)
	}
}
