package internal

import (
	"quantumcoin/blockchain"
	"sync"
)

// Mempool: Zincir dışı bekleyen işlemleri tutar
type Mempool struct {
	mu           sync.Mutex
	Transactions []*blockchain.Transaction
}

// NewMempool: Boş mempool oluşturur
func NewMempool() *Mempool {
	return &Mempool{
		Transactions: []*blockchain.Transaction{},
	}
}

// Add: Yeni işlem ekle
func (mp *Mempool) Add(tx *blockchain.Transaction) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.Transactions = append(mp.Transactions, tx)
}

// GetTransactions: Bekleyen işlemleri getir (opsiyonel: deep copy)
func (mp *Mempool) GetTransactions() []*blockchain.Transaction {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	txs := make([]*blockchain.Transaction, len(mp.Transactions))
	copy(txs, mp.Transactions)
	return txs
}

// Clear: Tüm işlemleri sil (blok kazıldığında çağrılır)
func (mp *Mempool) Clear() {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.Transactions = []*blockchain.Transaction{}
}

// Opsiyonel: Belirli tx’i çıkar
// func (mp *Mempool) RemoveTx(id []byte) {...}
