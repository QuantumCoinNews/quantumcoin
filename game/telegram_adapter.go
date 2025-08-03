package game

import "fmt"

// gs: GameState yapısı (global veya parametre olarak alınır)
// player: Oyuncu adresi veya kullanıcı adı
// score: Eklenecek puan

func HandleTelegramScore(gs *GameState, player string, score int) {
	fmt.Printf("[Telegram] %s adlı oyuncunun skoru: %d\n", player, score)
	gs.AddScore(player, score)

	// Oyun içi başarıya göre ödül sistemi
	if score > 1000 {
		GiveReward(player, "HighScore", 10)
	}
}
