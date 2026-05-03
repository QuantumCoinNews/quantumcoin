package blockchain

import "quantumcoin/config"

const (
	difficultyRetargetWindow = 100
	maxDifficultyBits        = 255
)

// NextDifficulty mevcut zincir temposuna göre önerilen PoW difficulty değerini döndürür.
// defaultBits minimum/taban difficulty olarak kullanılır.
// Mantık:
// - Yeterli blok yoksa defaultBits döner.
// - Son 100 blok hedef süreden hızlıysa difficulty +1.
// - Son 100 blok hedef süreden çok yavaşsa difficulty -1.
// - Difficulty hiçbir zaman defaultBits altına düşmez.
func (bc *Blockchain) NextDifficulty(defaultBits int) int {
	if defaultBits <= 0 {
		defaultBits = 16
	}
	if defaultBits > maxDifficultyBits {
		defaultBits = maxDifficultyBits
	}

	if bc == nil || len(bc.Blocks) <= difficultyRetargetWindow {
		return defaultBits
	}

	last := bc.Blocks[len(bc.Blocks)-1]
	prev := bc.Blocks[len(bc.Blocks)-1-difficultyRetargetWindow]
	if last == nil || prev == nil {
		return defaultBits
	}

	current := last.Difficulty
	if current <= 0 {
		current = defaultBits
	}
	if current < defaultBits {
		current = defaultBits
	}
	if current > maxDifficultyBits {
		current = maxDifficultyBits
	}

	targetSecs := config.Current().TargetBlockTimeSecs
	if targetSecs <= 0 {
		targetSecs = 30
	}

	expected := int64(targetSecs * difficultyRetargetWindow)
	actual := last.Timestamp - prev.Timestamp
	if actual <= 0 || expected <= 0 {
		return current
	}

	// Çok hızlıysa zorluğu artır.
	if actual < expected*8/10 && current < maxDifficultyBits {
		return current + 1
	}

	// Çok yavaşsa zorluğu azalt ama defaultBits altına düşürme.
	if actual > expected*12/10 && current > defaultBits {
		return current - 1
	}

	return current
}
