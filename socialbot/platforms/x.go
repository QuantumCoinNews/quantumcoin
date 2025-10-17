package main

import "log"

// Şimdilik sadece log basıyor (stub)
func PublishX(item ContentItem) error {
	log.Printf("[stub] X paylaşımı: %s", item.Title)
	return nil
}
