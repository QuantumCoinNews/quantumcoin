package token

import (
	"errors"
	"sync"
)

type QC20Token struct {
	Name        string
	Symbol      string
	TotalSupply uint64
	Decimals    uint8
	Balances    map[string]uint64
	Allowances  map[string]map[string]uint64
	mu          sync.RWMutex
}

// Yeni token oluştur
func NewQC20Token(name, symbol string, supply uint64, decimals uint8) *QC20Token {
	token := &QC20Token{
		Name:        name,
		Symbol:      symbol,
		TotalSupply: supply,
		Decimals:    decimals,
		Balances:    make(map[string]uint64),
		Allowances:  make(map[string]map[string]uint64),
	}
	// Token yaratılırken initial supply'ı yaratan adrese yazabilirsin
	// token.Balances[creator] = supply
	return token
}

// Mint: Yeni token basımı
func (t *QC20Token) Mint(to string, amount uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Balances[to] += amount
	t.TotalSupply += amount
}

// Burn: Token yak (isteğe bağlı, ileri seviye)
func (t *QC20Token) Burn(from string, amount uint64) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.Balances[from] < amount {
		return errors.New("yetersiz bakiye")
	}
	t.Balances[from] -= amount
	t.TotalSupply -= amount
	return nil
}

// BalanceOf: Cüzdan bakiyesi
func (t *QC20Token) BalanceOf(addr string) uint64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Balances[addr]
}

// Transfer: Klasik token transferi
func (t *QC20Token) Transfer(from, to string, amount uint64) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.Balances[from] < amount {
		return errors.New("yetersiz bakiye")
	}
	t.Balances[from] -= amount
	t.Balances[to] += amount
	return nil
}

// Approve: Harcama izni ver
func (t *QC20Token) Approve(owner, spender string, amount uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.Allowances[owner]; !ok {
		t.Allowances[owner] = make(map[string]uint64)
	}
	t.Allowances[owner][spender] = amount
}

// TransferFrom: Onayla transfer
func (t *QC20Token) TransferFrom(owner, spender, recipient string, amount uint64) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	allowed := t.Allowances[owner][spender]
	if allowed < amount {
		return errors.New("izin yetersiz")
	}
	if t.Balances[owner] < amount {
		return errors.New("bakiye yetersiz")
	}
	t.Balances[owner] -= amount
	t.Balances[recipient] += amount
	t.Allowances[owner][spender] -= amount
	return nil
}

// Allowance: Harcama izni miktarı
func (t *QC20Token) Allowance(owner, spender string) uint64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Allowances[owner][spender]
}
