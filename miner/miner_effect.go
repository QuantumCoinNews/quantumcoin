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
	return &Effect{symbol: symbol, last: time.Now()}
}

// Frame: terminalde hafif animasyon (CPU dostu)
func (e *Effect) Frame(step, nextHeight, difficulty int, hashrate float64) {
	spin := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}[step%10]
	bar := progressBar(step, 24)
	line := fmt.Sprintf("\r%s %s Mining h=%d diff=%d %s  ~%.2f H/s",
		spin, e.symbol, nextHeight, difficulty, bar, hashrate)
	fmt.Print(line)
}

func (e *Effect) Clear() {
	fmt.Print("\r" + strings.Repeat(" ", 90) + "\r")
}

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
