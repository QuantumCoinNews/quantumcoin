package game

import "fmt"

// HandleTelegramScore: Telegram’dan gelen skorları işler (örnek adaptör)
func HandleTelegramScore(gs *GameState, player string, score int) {
	if gs == nil || player == "" || score == 0 {
		return
	}
	fmt.Printf("[Telegram] %s adlı oyuncunun skoru: %+d\n", player, score)
	gs.AddScore(player, score)

	// Skor güncellendi, eşiklere göre bonus ver
	EvaluateAndReward(gs, player)

	// Kalıcılık açıksa (dosya set edildiyse) sessizce kaydet
	_ = gs.Save("")
}
