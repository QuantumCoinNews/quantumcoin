package ai

import "strings"

// normalizeBonusReason:
// Bonus açıklamalarını standartlaştırmak için ortak helper.
// bonus_builder.go ve diğer yerler burayı kullanır.
func normalizeBonusReason(reason string) string {
	r := strings.TrimSpace(reason)
	if r == "" {
		return "AI bonus"
	}

	low := strings.ToLower(r)

	if strings.Contains(low, "high frequency") {
		// Çok sık + yüksek hacimli işlemler için daha düzgün bir mesaj
		return "frequency + volume based bonus"
	}

	if strings.Contains(low, "large transaction") {
		// Büyük/alışılmışın dışındaki işlemler için eğitim amaçlı bonus
		return "AI: suspicious pattern education bonus"
	}

	// Hiçbiri değilse gelen metni bozmadan geri döndür
	return r
}
