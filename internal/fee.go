package internal

import (
	"sync"

	"quantumcoin/blockchain"
)

// Varsayılan sabit ücret (QC cinsinden). İstersen SetFixedFee ile değiştirebilirsin.
const defaultFixedFee = 1

type FeeManager struct {
	mu        sync.Mutex
	fixedFee  int
	totalFees int
}

func NewFeeManager() *FeeManager {
	return &FeeManager{fixedFee: defaultFixedFee}
}

// SetFixedFee: runtime'da ücreti günceller (>=0 olmalı)
func (fm *FeeManager) SetFixedFee(fee int) {
	if fee < 0 {
		fee = 0
	}
	fm.mu.Lock()
	fm.fixedFee = fee
	fm.mu.Unlock()
}

// GetFixedFee: geçerli sabit ücreti döndürür
func (fm *FeeManager) GetFixedFee() int {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	return fm.fixedFee
}

// ApplyFee: bir işleme ücret uygular ve toplam ücreti artırır.
// Şimdilik sabit ücret modeli; ileride tx boyutu/önceliğe göre genişletilebilir.
func (fm *FeeManager) ApplyFee(_ *blockchain.Transaction) int {
	fm.mu.Lock()
	fee := fm.fixedFee
	fm.totalFees += fee
	fm.mu.Unlock()
	return fee
}

// GetTotalFees: toplanmış toplam ücreti döndürür
func (fm *FeeManager) GetTotalFees() int {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	return fm.totalFees
}

// Reset: toplam ücret sayacını sıfırlar
func (fm *FeeManager) Reset() {
	fm.mu.Lock()
	fm.totalFees = 0
	fm.mu.Unlock()
}
