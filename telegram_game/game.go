package telegram_game

import "log"

// NotifyBlockMined: zincirde yeni blok üretildiğinde Telegram oyun entegrasyonuna bildirim.
// İmza: (height, hash, miner)
func NotifyBlockMined(height int, hash string, miner string) {
	log.Printf("[TelegramGame] Block mined: height=%d hash=%s miner=%s", height, hash, miner)
}
