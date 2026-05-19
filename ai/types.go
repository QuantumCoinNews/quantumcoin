// ai/types.go
package ai

import "time"

// tx'leri hafifletilmiş halde AI'ye veriyoruz
type TxLite struct {
	TxID          string    `json:"tx_id"`
	WalletAddress string    `json:"wallet_address"`
	Sender        string    `json:"sender"`
	Amount        float64   `json:"amount"`
	Timestamp     time.Time `json:"timestamp"`
}

// analyzer'ın ürettiği kayıt
type AnomalyReport struct {
	WalletAddress string  `json:"wallet_address"`
	TxID          string  `json:"tx_id"`
	Suspicious    bool    `json:"suspicious"`
	Score         float64 `json:"score"`
	Reason        string  `json:"reason"`
}

// bonus üretirken kullandığımız yapı
type BonusSuggestion struct {
	WalletAddress string `json:"wallet_address"`
	Source        string `json:"source"`
	Amount        int    `json:"amount"`
	Reason        string `json:"reason"`
}

// 🔴 senin recommendation.go burada bunu istiyor
type Recommendation struct {
	WalletAddress string  `json:"wallet_address"`
	Action        string  `json:"action"`  // monitor / educate-user ...
	Reason        string  `json:"reason"`  // anomaly'den gelen sebep
	Message       string  `json:"message"` // kullanıcıya gösterilecek metin
	Score         float64 `json:"score"`
}

// ödül optimizasyonu kullanıyorsan bu da dursun
type RewardSuggestion struct {
	WalletAddress   string  `json:"wallet_address"`
	SuggestedReward float64 `json:"suggested_reward"`
	Reason          string  `json:"reason"`
}
