package game

import (
	"sort"
)

// PlayerScore: liderlik tablosu için DTO
type PlayerScore struct {
	Player string `json:"player"`
	Score  int    `json:"score"`
}

// GetTopPlayers: en yüksek N skor
func GetTopPlayers(gs *GameState, n int) []PlayerScore {
	if gs == nil || n <= 0 {
		return []PlayerScore{}
	}

	// snapshot
	gs.mu.RLock()
	list := make([]PlayerScore, 0, len(gs.scores))
	for p, s := range gs.scores {
		list = append(list, PlayerScore{Player: p, Score: s})
	}
	gs.mu.RUnlock()

	sort.Slice(list, func(i, j int) bool {
		if list[i].Score == list[j].Score {
			return list[i].Player < list[j].Player
		}
		return list[i].Score > list[j].Score
	})
	if n > len(list) {
		n = len(list)
	}
	return list[:n]
}
