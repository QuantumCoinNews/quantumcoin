// Package game contains QuantumCoin project code for this module.
package game

import (
	"sync"
)

// GameState documents the QuantumCoin component used by this module.
// GameState: thread-safe basit skor deposu
type GameState struct {
	mu     sync.RWMutex
	scores map[string]int // player -> score
}

// NewGameState documents the QuantumCoin component used by this module.
// NewGameState: başlangıç
func NewGameState() *GameState {
	return &GameState{
		scores: make(map[string]int),
	}
}

// AddScore documents the QuantumCoin component used by this module.
// AddScore: oyuncu skoruna delta ekle (negatif olabilir)
func (gs *GameState) AddScore(player string, delta int) {
	if gs == nil || player == "" || delta == 0 {
		return
	}
	gs.mu.Lock()
	gs.scores[player] += delta
	gs.mu.Unlock()
}

// GetScore documents the QuantumCoin component used by this module.
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
