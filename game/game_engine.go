package game

import (
	"sync"
)

// GameState: thread-safe basit skor deposu
type GameState struct {
	mu     sync.RWMutex
	scores map[string]int // player -> score
}

// NewGameState: başlangıç
func NewGameState() *GameState {
	return &GameState{
		scores: make(map[string]int),
	}
}

// AddScore: oyuncu skoruna delta ekle (negatif olabilir)
func (gs *GameState) AddScore(player string, delta int) {
	if gs == nil || player == "" || delta == 0 {
		return
	}
	gs.mu.Lock()
	gs.scores[player] += delta
	gs.mu.Unlock()
}

// GetScore: oyuncu mevcut skoru
func (gs *GameState) GetScore(player string) int {
	if gs == nil || player == "" {
		return 0
	}
	gs.mu.RLock()
	score := gs.scores[player]
	gs.mu.RUnlock()
	return score
}
