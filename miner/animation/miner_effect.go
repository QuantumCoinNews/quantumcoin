// animation/miner_effect.go
package animation

import (
	"fmt"
	"strings"
	"time"
)

// ====== ANSI renkler / kontrol ======
const (
	clrReset = "\033[0m"
	clrDim   = "\033[2m"
	clrCyan  = "\033[36m"
	clrGreen = "\033[32m"
	clrGold  = "\033[33m"
	clrPink  = "\033[95m"
	clrBlue  = "\033[34m"
	clrRed   = "\033[31m"

	escClear = "\033[2J"
	escHome  = "\033[H"
)

// Temizle
func ClearConsole() {
	fmt.Print(escClear)
	fmt.Print(escHome)
}

// Basit progress bar
func progressBar(ratio float64, width int) string {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	if width <= 0 {
		width = 10
	}
	fill := int(ratio * float64(width))
	if fill < 0 {
		fill = 0
	}
	if fill > width {
		fill = width
	}
	return "[" + strings.Repeat("█", fill) + strings.Repeat("░", width-fill) + "]"
}

var spinner = []string{"⠋", "⠙", "⠸", "⠴", "⠦", "⠇"}

// ====== Canlı kazım karesi ======
// *miner.go* bu imzayı çağırıyor: ShowMiningFrame(step, width, startH, nextH, diff)
func ShowMiningFrame(step, width, _startH, nextH, diff int) {
	if width <= 0 {
		width = 10
	}
	// width'ü adım toplamı gibi kullanıyoruz
	ratio := float64(step%(width+1)) / float64(width)

	ClearConsole()
	fmt.Printf("%s%s QuantumCoin Miner %s\n", clrCyan, spinner[step%len(spinner)], clrReset)
	fmt.Printf("%sHeight:%s %d   %sDiff:%s %d bits\n",
		clrDim, clrReset, nextH,
		clrDim, clrReset, diff,
	)
	fmt.Println()
	fmt.Printf("%s Mining block %s%s\n", clrBlue, progressBar(ratio, 26), clrReset)
	// ufak bekleme: kare hızını kontrol
	time.Sleep(120 * time.Millisecond)
}

// ====== Kısa otomatik animasyon (geri uyumluluk) ======
func ShowMiningAnimation() {
	for i := 0; i < 11; i++ {
		// stepsTotal yerine width kullanıyoruz
		ShowMiningFrame(i, 10, 0, 0, 0)
	}
}

// ====== Ödül efekti ======
func ShowRewardEffect(amount float64) {
	ClearConsole()
	fmt.Println(clrGreen + "🎉  Block Found!" + clrReset)
	time.Sleep(120 * time.Millisecond)
	fmt.Println(clrGold + "🪙  Reward credited:" + clrReset)
	time.Sleep(120 * time.Millisecond)
	fmt.Printf("%s+ %.2f QC%s\n", clrGold, amount, clrReset)
	time.Sleep(200 * time.Millisecond)

	// mini konfeti
	conf := []string{"✨", "🎊", "🎉", "💥", "✨", "🎊"}
	for i := 0; i < len(conf); i++ {
		fmt.Printf("%s%s%s ", clrPink, conf[i], clrReset)
		time.Sleep(80 * time.Millisecond)
	}
	fmt.Println()
	time.Sleep(300 * time.Millisecond)
}

// ====== Ödül bölüşümü bilgi ekranı ======
func ShowSplitInfo(miner, stake, dev, burn, community float64) {
	fmt.Println()
	fmt.Printf("%sReward Split%s\n", clrDim, clrReset)
	fmt.Printf("  Miner      : %s%.2f QC%s\n", clrGreen, miner, clrReset)
	fmt.Printf("  Stake Pool : %s%.2f QC%s\n", clrCyan, stake, clrReset)
	fmt.Printf("  Dev Fund   : %s%.2f QC%s\n", clrBlue, dev, clrReset)
	fmt.Printf("  Burn       : %s%.2f QC%s\n", clrRed, burn, clrReset)
	fmt.Printf("  Community  : %s%.2f QC%s\n", clrGold, community, clrReset)

	// Küçük bar gösterimi
	type item struct {
		name  string
		val   float64
		color string
	}
	parts := []item{
		{"Miner     ", miner, clrGreen},
		{"Stake     ", stake, clrCyan},
		{"Dev       ", dev, clrBlue},
		{"Burn      ", burn, clrRed},
		{"Community ", community, clrGold},
	}
	total := miner + stake + dev + burn + community
	if total <= 0 {
		total = 1
	}
	const maxBars = 30.0
	for _, p := range parts {
		n := int((p.val / total) * maxBars)
		if n < 0 {
			n = 0
		}
		bar := strings.Repeat("█", n)
		fmt.Printf("  %s: %s%-30s%s  %.2f QC\n", p.name, p.color, bar, clrReset, p.val)
	}
	time.Sleep(900 * time.Millisecond)
}

// ====== Bonus/NFT parıltı ======
func ShowSparkle(event string) {
	ClearConsole()
	fmt.Println("✨✨✨")
	fmt.Printf("%sLucky Event:%s %s\n", clrGold, clrReset, event)
	fmt.Println("✨✨✨")
	time.Sleep(900 * time.Millisecond)
}
