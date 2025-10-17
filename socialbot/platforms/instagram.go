package main

import "log"

// Şimdilik sadece log basıyor (stub)
func PublishInstagram(item ContentItem) error {
	log.Printf("[stub] Instagram paylaşımı: %s", item.Title)
	return nil
}
