package game

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
)

const defaultGameFile = "game_store.json"

// GameState; oyuncu skorlarını ve (opsiyonel) son güncelleme zamanlarını tutar
type GameState struct {
	mu       sync.Mutex
	Players  map[string]int       `json:"players"`
	Updated  map[string]time.Time `json:"updated,omitempty"`
	filename string               // kalıcılık için dosya yolu (ops.)
}

// NewGameState: boş state (dosya yolu opsiyonel)
func NewGameState() *GameState {
	return &GameState{
		Players: make(map[string]int),
		Updated: make(map[string]time.Time),
	}
}

// WithFile: kalıcılık dosyası atanmış state döndürür
func (g *GameState) WithFile(path string) *GameState {
	g.mu.Lock()
	defer g.mu.Unlock()
	if path == "" {
		path = defaultGameFile
	}
	g.filename = path
	return g
}

// AddScore: oyuncuya skor ekler (negatifse 0 altına düşürmez)
func (g *GameState) AddScore(player string, score int) {
	if player == "" || score == 0 {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Players[player] += score
	if g.Players[player] < 0 {
		g.Players[player] = 0
	}
	g.Updated[player] = time.Now()
	fmt.Printf("%s kullanıcısına %d puan eklendi. Yeni skor: %d\n", player, score, g.Players[player])
}

// SetScore: doğrudan skor atar
func (g *GameState) SetScore(player string, score int) {
	if player == "" || score < 0 {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Players[player] = score
	g.Updated[player] = time.Now()
}

// GetScore: tek oyuncu skoru
func (g *GameState) GetScore(player string) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.Players[player]
}

// Snapshot: kilit dışı okuma için Players/Updated kopyası
func (g *GameState) Snapshot() (map[string]int, map[string]time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	pm := make(map[string]int, len(g.Players))
	um := make(map[string]time.Time, len(g.Updated))
	for k, v := range g.Players {
		pm[k] = v
	}
	for k, v := range g.Updated {
		um[k] = v
	}
	return pm, um
}

// Reset: tüm skorları temizler
func (g *GameState) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Players = make(map[string]int)
	g.Updated = make(map[string]time.Time)
}

// Save: JSON olarak diske yazar
func (g *GameState) Save(path string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if path == "" {
		if g.filename == "" {
			g.filename = defaultGameFile
		}
		path = g.filename
	}
	b, err := json.MarshalIndent(struct {
		Players map[string]int       `json:"players"`
		Updated map[string]time.Time `json:"updated,omitempty"`
	}{
		Players: g.Players,
		Updated: g.Updated,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

// Load: JSON’dan yükler (varsa)
func (g *GameState) Load(path string) error {
	if path == "" {
		if g.filename == "" {
			g.filename = defaultGameFile
		}
		path = g.filename
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var tmp struct {
		Players map[string]int       `json:"players"`
		Updated map[string]time.Time `json:"updated,omitempty"`
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if tmp.Players == nil {
		tmp.Players = map[string]int{}
	}
	if tmp.Updated == nil {
		tmp.Updated = map[string]time.Time{}
	}
	g.Players = tmp.Players
	g.Updated = tmp.Updated
	return nil
}

// Sorted: skorları büyükten küçüğe, eşitse isme göre ASC sıralı dilim döndürür
func (g *GameState) Sorted() []struct {
	Player string
	Score  int
} {
	pm, _ := g.Snapshot()
	arr := make([]struct {
		Player string
		Score  int
	}, 0, len(pm))
	for p, s := range pm {
		arr = append(arr, struct {
			Player string
			Score  int
		}{Player: p, Score: s})
	}
	sort.Slice(arr, func(i, j int) bool {
		if arr[i].Score == arr[j].Score {
			return arr[i].Player < arr[j].Player
		}
		return arr[i].Score > arr[j].Score
	})
	return arr
}
