package main

import "log"

// Şimdilik sadece log basıyor (stub)
func PublishYouTube(item ContentItem) error {
	log.Printf("[stub] YouTube paylaşımı: %s", item.Title)
	return nil
}
