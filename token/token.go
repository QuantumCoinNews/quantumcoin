package token

import (
	"errors"
	"sync"
)

// Token: basit QC-20
type Token struct {
	Symbol      string            `json:"symbol"`
	Name        string            `json:"name"`
	Decimals    uint8             `json:"decimals"`
	MaxSupply   uint64            `json:"max_supply"`
	TotalSupply uint64            `json:"total_supply"`
	Owner       string            `json:"owner"`
	Balances    map[string]uint64 `json:"balances"`
	mu          sync.Mutex        `json:"-"`
}

func New(symbol, name string, decimals uint8, maxSupply uint64, owner string) *Token {
	return &Token{
		Symbol:    symbol,
		Name:      name,
		Decimals:  decimals,
		MaxSupply: maxSupply,
		Owner:     owner,
		Balances:  make(map[string]uint64),
	}
}

func (t *Token) Mint(caller, to string, amount uint64) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if caller != t.Owner {
		return errors.New("not token owner")
	}
	if amount == 0 {
		return errors.New("amount is zero")
	}
	if t.MaxSupply > 0 && t.TotalSupply+amount > t.MaxSupply {
		return errors.New("max supply exceeded")
	}
	t.Balances[to] += amount
	t.TotalSupply += amount
	return nil
}

func (t *Token) Transfer(from, to string, amount uint64) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if amount == 0 {
		return errors.New("amount is zero")
	}
	if t.Balances[from] < amount {
		return errors.New("insufficient balance")
	}
	t.Balances[from] -= amount
	t.Balances[to] += amount
	return nil
}

func (t *Token) BalanceOf(addr string) uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Balances[addr]
}

func (t *Token) GetTotalSupply() uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.TotalSupply
}
