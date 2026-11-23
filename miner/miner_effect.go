package miner

import (
	"fmt"
	"strings"
	"time"
)

type Effect struct {
	symbol string
	last   time.Time
}

func NewEffect(symbol string) *Effect {
	return &Effect{symbol: symbol}
}

// Frame documents the QuantumCoin component used by this module.
// Frame: terminalde hafif animasyon (CPU dostu)
func (e *Effect) Frame(step, nextHeight, difficulty int, hashrate float64) {
	now := time.Now()
	if now.Sub(e.last) < time.Second {
		return
	}
	e.last = now
	spin := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}[step%10]
	bar := progressBar(step, 24)
	fmt.Printf("%s %s Mining h=%d diff=%d %s  ~%.2f H/s\n",
		spin, e.symbol, nextHeight, difficulty, bar, hashrate)
}

// Clear is retained for callers of the previous in-place display.
func (e *Effect) Clear() {}

func progressBar(step, width int) string {
	pos := step % width
	sb := strings.Builder{}
	sb.Grow(width + 2)
	sb.WriteByte('[')
	for i := 0; i < width; i++ {
		if i == pos {
			sb.WriteString("█")
		} else {
			sb.WriteString("·")
		}
	}
	sb.WriteByte(']')
	return sb.String()
}
