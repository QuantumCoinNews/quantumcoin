package game

import (
	"fmt"
	"sync"
)

type GameState struct {
	Players map[string]int // Adres:Skor eşlemesi
	mu      sync.Mutex     // Thread-safe
}

func NewGameState() *GameState {
	return &GameState{Players: make(map[string]int)}
}

func (g *GameState) AddScore(player string, score int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Players[player] += score
	fmt.Printf("%s kullanıcısına %d puan eklendi.\n", player, score)
}
