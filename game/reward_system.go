package game

import (
	"fmt"
	"quantumcoin/internal"
)

// Oyuncuya zincir üstü ödül/bounus verme fonksiyonu
func GiveReward(player string, rewardType string, amount int) {
	fmt.Printf("%s kullanıcısına %d adet '%s' ödülü verildi.\n", player, amount, rewardType)
	// Zincir üstü bonus entegrasyonu!
	internal.GiveBonus(player, internal.BonusTypeEvent, amount, rewardType, "")
}
