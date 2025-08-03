package socialbot

import (
	"fmt"
	"time"
)

// Otomatik içerik üretimi (topic + random varyasyon ile)
// AI veya LLM entegrasyonu ileride kolayca eklenebilir!
func GenerateContent(topic string) (caption, filePath string) {
	fmt.Printf("Otomatik içerik üretiliyor: %s\n", topic)
	// Şu an mock veri, istersen AI API/Stable Diffusion/LLM bağlayabilirsin
	caption = fmt.Sprintf("Otomatik gönderi: %s | %d", topic, randInt())
	filePath = "/tmp/otomatik-fake-media.jpg"
	return
}

func randInt() int {
	return int(time.Now().UnixNano() % 100000)
}
