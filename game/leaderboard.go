package game

import (
	"fmt"
	"sort"
)

type LeaderboardEntry struct {
	Player string
	Score  int
}

// Basit skor sıralama fonksiyonu
func GetTopPlayers(gameState *GameState, limit int) []LeaderboardEntry {
	gameState.mu.Lock()
	defer gameState.mu.Unlock()
	var entries []LeaderboardEntry
	for player, score := range gameState.Players {
		entries = append(entries, LeaderboardEntry{Player: player, Score: score})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Score > entries[j].Score
	})
	if len(entries) > limit {
		return entries[:limit]
	}
	return entries
}

func PrintLeaderboard(lb []LeaderboardEntry) {
	fmt.Println("=== Liderlik Tablosu ===")
	for i, entry := range lb {
		fmt.Printf("%d. %s - %d puan\n", i+1, entry.Player, entry.Score)
	}
}
