package animation

import (
	"fmt"
	"time"
)

// ClearConsole temizleme efekti (Windows ve Linux için uyarlanabilir)
func ClearConsole() {
	fmt.Print("\033[2J")
	fmt.Print("\033[H")
}

// ShowMiningAnimation basit bir kazma animasyonu
func ShowMiningAnimation() {
	frames := []string{
		`[⚒] Mining block ░░░░░░░░░░`,
		`[⚒] Mining block █░░░░░░░░░`,
		`[⚒] Mining block ██░░░░░░░░`,
		`[⚒] Mining block ███░░░░░░░`,
		`[⚒] Mining block ████░░░░░░`,
		`[⚒] Mining block █████░░░░░`,
		`[⚒] Mining block ██████░░░░`,
		`[⚒] Mining block ███████░░░`,
		`[⚒] Mining block ████████░░`,
		`[⚒] Mining block █████████░`,
		`[⚒] Mining block ██████████`,
	}

	for _, frame := range frames {
		ClearConsole()
		fmt.Println(frame)
		time.Sleep(300 * time.Millisecond)
	}
}

// ShowRewardEffect blok ödülü animasyonu
func ShowRewardEffect(amount float64) {
	ClearConsole()
	fmt.Println("🎉🎉🎉")
	fmt.Printf("Congratulations! You earned %.2f QC 🪙\n", amount)
	fmt.Println("🎉🎉🎉")
	time.Sleep(2 * time.Second)
}

// ShowSparkle animasyonu (NFT veya bonus için)
func ShowSparkle(event string) {
	ClearConsole()
	fmt.Println("✨✨✨")
	fmt.Printf("Lucky Event: %s!\n", event)
	fmt.Println("✨✨✨")
	time.Sleep(2 * time.Second)
}
