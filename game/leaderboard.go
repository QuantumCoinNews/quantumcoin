package game

import (
	"fmt"
)

// LeaderboardEntry: dış API’de kullanılan tip
type LeaderboardEntry struct {
	Player string `json:"player"`
	Score  int    `json:"score"`
}

// GetTopPlayers: en iyi `limit` oyuncu
func GetTopPlayers(gameState *GameState, limit int) []LeaderboardEntry {
	if gameState == nil || limit <= 0 {
		return nil
	}
	arr := gameState.Sorted()
	if len(arr) > limit {
		arr = arr[:limit]
	}
	out := make([]LeaderboardEntry, 0, len(arr))
	for _, e := range arr {
		out = append(out, LeaderboardEntry{Player: e.Player, Score: e.Score})
	}
	return out
}

// PrintLeaderboard: konsola yazdırır
func PrintLeaderboard(lb []LeaderboardEntry) {
	fmt.Println("=== Liderlik Tablosu ===")
	for i, entry := range lb {
		fmt.Printf("%d. %s - %d puan\n", i+1, entry.Player, entry.Score)
	}
}
