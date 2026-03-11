// internal/mempool.go
package internal

import (
	"bytes"
	"encoding/hex"
	"sync"

	"quantumcoin/ai"
	"quantumcoin/blockchain"
)

// Mempool: Zincir dışı bekleyen işlemleri tutar
type Mempool struct {
	mu           sync.Mutex
	transactions []*blockchain.Transaction
	index        map[string]struct{} // hızlı "var mı?" kontrolü için (txID raw)
	capacity     int                 // 0 veya negatifse sınırsız
}

// NewMempool: Boş mempool oluşturur
func NewMempool() *Mempool {
	return &Mempool{
		transactions: []*blockchain.Transaction{},
		index:        make(map[string]struct{}),
		capacity:     0, // sınırsız (istersen dışarıdan SetCapacity ile kısıtlayabilirsin)
	}
}

// SetCapacity: maksimum bekleyen işlem sayısı (0/sıfır veya <0 => sınırsız)
func (mp *Mempool) SetCapacity(n int) {
	mp.mu.Lock()
	mp.capacity = n
	mp.mu.Unlock()
}

// Len: mempool boyutu
func (mp *Mempool) Len() int {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	return len(mp.transactions)
}

// Has: tx ID mevcut mu?
func (mp *Mempool) Has(txID []byte) bool {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	_, ok := mp.index[string(txID)]
	return ok
}

// Add: Yeni işlem ekle (tekrarları engeller, kapasiteyi uygular, imza + AI filtresi uygular)
func (mp *Mempool) Add(tx *blockchain.Transaction) bool {
	if tx == nil || len(tx.ID) == 0 {
		return false
	}

	// 1) İmza doğrulama (coinbase hariç her şey imzalı olsun)
	if !tx.IsCoinbase() && !tx.Verify() {
		return false
	}

	key := string(tx.ID)

	mp.mu.Lock()
	defer mp.mu.Unlock()

	// 2) kapasite kontrolü
	if mp.capacity > 0 && len(mp.transactions) >= mp.capacity {
		return false
	}

	// 3) tekrar kontrolü
	if _, ok := mp.index[key]; ok {
		return false
	}

	// 4) === AI PREFILTER ===
	if ai.Enabled() && tx.Sender != "" {
		lite := []ai.TxLite{{
			TxID:          hex.EncodeToString(tx.ID),
			WalletAddress: tx.Sender,
			Sender:        tx.Sender,
			Amount:        tx.Amount,
			Timestamp:     tx.Timestamp,
		}}

		// ai/analyzer.go: func AnalyzeTransactions(txs []TxLite) []AnomalyReport
		reports := ai.AnalyzeTransactions(lite)
		if len(reports) > 0 && reports[0].Suspicious {
			// şüpheli işlem mempool'a alınmaz
			return false
		}
	}
	// === /AI PREFILTER ===

	mp.transactions = append(mp.transactions, tx)
	mp.index[key] = struct{}{}
	return true
}

// GetTransactions: Bekleyen işlemleri kopyalayarak döner
func (mp *Mempool) GetTransactions() []*blockchain.Transaction {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	txs := make([]*blockchain.Transaction, len(mp.transactions))
	copy(txs, mp.transactions)
	return txs
}

// PopBatch: en fazla 'n' işlemi FIFO olarak çıkarır ve döner
func (mp *Mempool) PopBatch(n int) []*blockchain.Transaction {
	if n <= 0 {
		return nil
	}
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if n > len(mp.transactions) {
		n = len(mp.transactions)
	}
	batch := mp.transactions[:n]

	// index'ten sil
	for _, tx := range batch {
		delete(mp.index, string(tx.ID))
	}

	// geri kalan
	rest := make([]*blockchain.Transaction, len(mp.transactions)-n)
	copy(rest, mp.transactions[n:])
	mp.transactions = rest

	return batch
}

// RemoveTx: belirli txID'yi mempool'dan siler (varsa)
func (mp *Mempool) RemoveTx(txID []byte) bool {
	key := string(txID)
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if _, ok := mp.index[key]; !ok {
		return false
	}
	// slice içinden bul & çıkar
	for i, tx := range mp.transactions {
		if bytes.Equal(tx.ID, txID) {
			mp.transactions = append(mp.transactions[:i], mp.transactions[i+1:]...)
			delete(mp.index, key)
			return true
		}
	}
	// bulamazsa indeksi temizle (tutarlılık)
	delete(mp.index, key)
	return false
}

// Clear: Tüm işlemleri sil (blok kazıldığında çağrılır)
func (mp *Mempool) Clear() {
	mp.mu.Lock()
	mp.transactions = []*blockchain.Transaction{}
	mp.index = make(map[string]struct{})
	mp.mu.Unlock()
}
