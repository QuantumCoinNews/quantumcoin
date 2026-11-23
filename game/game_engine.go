// Package game retains historical signed-score projection compatibility.
// Authenticated Telegram SQLite services own runtime gameplay and rewards.
package game

import (
	"errors"
	"sync"
)

// GameState is an advisory projection cache, never a gameplay/economic ledger.
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

// AddScore is a retired demo API. Only verified signed projections may update this cache.
// Deprecated: gameplay is owned by the authenticated Telegram SQLite backend.
func (gs *GameState) AddScore(player string, delta int) error { return ErrLegacyGameDisabled }

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

var ErrLegacyGameDisabled = errors.New("legacy Go gameplay/rewards disabled; use authenticated Telegram backend")
