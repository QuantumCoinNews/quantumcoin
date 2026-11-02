// Package game contains QuantumCoin project code for this module.
package game

// HandleTelegramScore documents the QuantumCoin component used by this module.
// HandleTelegramScore: dış sistemlerden gelen skor güncellemesi (delta)
func HandleTelegramScore(gs *GameState, player string, score int) {
	if gs == nil || player == "" {
		return
	}
	gs.AddScore(player, score)
}
