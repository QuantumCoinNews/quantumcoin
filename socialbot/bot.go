package socialbot

import (
	"fmt"
	"quantumcoin/socialbot/platforms" //
	"time"
)

type SocialBot struct{}

func (b *SocialBot) Start() {
	fmt.Println("SocialBot başlatıldı.")
	Start(b) // Scheduler ile otomatik başlar
}

// Otomatik içerik üretip, bütün platformlara paylaş
func (b *SocialBot) DoAutoShare() {
	topic := getTodayTopic() // Her gün farklı bir konu
	caption, filePath := GenerateContent(topic)

	fmt.Println("🟢 Otomatik paylaşım başlıyor:", topic)

	platforms.PostInstagramMedia(caption, filePath)
	platforms.PostTikTokVideo(caption, filePath)
	platforms.PostXStatus(caption, filePath)
	platforms.PostYouTubeVideo(topic, caption, filePath)
	fmt.Println("🟢 Tüm platformlarda paylaşım tamamlandı.")
}

// Her gün için farklı bir konu (AI, trend, random…)
func getTodayTopic() string {
	topics := []string{
		"Quantum Coin Haberleri",
		"Blokzincir Eğitimi",
		"Kripto Analizleri",
		"Quantum Mining İpuçları",
		"Haftanın NFT'si",
	}
	t := time.Now().Day() % len(topics)
	return topics[t]
}
