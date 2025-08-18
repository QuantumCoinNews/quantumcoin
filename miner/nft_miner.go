package miner

import (
	"fmt"
	"log"
	"time"

	"quantumcoin/blockchain"
)

// ---- Opsiyonel entegrasyon kancaları (DB/Telegram vb.) ----
// Bu hook'lar nil ise çağrılmaz; böylece dış paketlere ihtiyaç olmadan derlenir.
var SaveNFTRewardHook func(address, nftType, txID string, at time.Time) error
var NotifyTelegramHook func(address, message string)

// NFTMiningReward: nadir NFT ödül kaydı
type NFTMiningReward struct {
	Address string
	Type    string // "rare", "epic", "legendary" vb.
	Time    time.Time
	TxID    string // zincir üstü işlem ID'si (varsa)
}

// GrantNFTBonus: madenciye NFT ödülü verir (+ isteğe bağlı zincir/DB/Telegram bildirimleri)
func GrantNFTBonus(address, nftType string, bc *blockchain.Blockchain) {
	reward := NFTMiningReward{
		Address: address,
		Type:    nftType,
		Time:    time.Now(),
	}

	// Terminal çıktısı
	fmt.Printf("🎁 %s adresine '%s' türünde NFT ödülü verildi! (%s)\n",
		reward.Address, reward.Type, reward.Time.Format("02 Jan 2006 15:04:05"))

	// 1) Zincir üstü kayıt (stub; gerçek mint akışına bağlanacak)
	if bc != nil {
		txID, err := bc.MintNFT(reward.Address, reward.Type, map[string]string{
			"source": "mining_reward",
			"date":   reward.Time.Format(time.RFC3339),
		})
		if err != nil {
			log.Printf("[NFT] zincir üstü kayıt başarısız: %v", err)
		} else {
			reward.TxID = txID
			log.Printf("[NFT] zincir üstü kayıt tamamlandı, TxID: %s", txID)
		}
	}

	// 2) Veritabanı hook'u (opsiyonel)
	if SaveNFTRewardHook != nil {
		if err := SaveNFTRewardHook(reward.Address, reward.Type, reward.TxID, reward.Time); err != nil {
			log.Printf("[NFT] veritabanı kaydı başarısız: %v", err)
		}
	}

	// 3) Telegram bildirim hook'u (opsiyonel)
	if NotifyTelegramHook != nil {
		NotifyTelegramHook(reward.Address,
			fmt.Sprintf("🚀 Uzay madenciliğinde **%s** türünde NFT kazandınız! 🎉", reward.Type))
	}
}

// Basit rastgele NFT tipi
func GenerateRandomNFTType() string {
	types := []string{"rare", "epic", "legendary"}
	return types[time.Now().UnixNano()%int64(len(types))]
}
