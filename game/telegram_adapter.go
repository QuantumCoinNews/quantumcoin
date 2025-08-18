package game

// HandleTelegramScore: dış sistemlerden gelen skor güncellemesi (delta)
func HandleTelegramScore(gs *GameState, player string, score int) {
	if gs == nil || player == "" {
		return
	}
	gs.AddScore(player, score)
}
