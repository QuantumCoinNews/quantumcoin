package internal

import (
	"quantumcoin/blockchain"
)

// Sabit işlem ücreti (örnek): 0.00001 QC (1 satoshi)
const FixedFee = 1 // En küçük birim (int ile tutulur)

type FeeManager struct {
	TotalFees int // Blok/mempool için toplanan toplam fee
}

// Yeni FeeManager oluştur
func NewFeeManager() *FeeManager {
	return &FeeManager{TotalFees: 0}
}

// ApplyFee: İşleme fee uygula, toplam fee havuzuna ekle, madenciye aktarılacak kısmı döndür
func (fm *FeeManager) ApplyFee(tx *blockchain.Transaction) int {
	fm.TotalFees += FixedFee
	return FixedFee
}

// GetTotalFees: Blok/madenci için toplanan tüm fee
func (fm *FeeManager) GetTotalFees() int {
	return fm.TotalFees
}

// Reset: Fee havuzunu sıfırla (blok çıkarıldıktan sonra çağrılır)
func (fm *FeeManager) Reset() {
	fm.TotalFees = 0
}
