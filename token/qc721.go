package token

import (
	"errors"
	"sync"
)

type QC721Token struct {
	Name      string
	Symbol    string
	Owners    map[uint64]string
	TokenURIs map[uint64]string
	mu        sync.RWMutex
}

// Yeni NFT koleksiyonu başlat
func NewQC721Token(name, symbol string) *QC721Token {
	return &QC721Token{
		Name:      name,
		Symbol:    symbol,
		Owners:    make(map[uint64]string),
		TokenURIs: make(map[uint64]string),
	}
}

// NFT mint (oluştur ve sahip ata)
func (n *QC721Token) Mint(to string, tokenID uint64, tokenURI string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, exists := n.Owners[tokenID]; exists {
		return errors.New("token zaten var")
	}
	n.Owners[tokenID] = to
	n.TokenURIs[tokenID] = tokenURI
	return nil
}

// Sahibi kim?
func (n *QC721Token) OwnerOf(tokenID uint64) (string, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	owner, ok := n.Owners[tokenID]
	if !ok {
		return "", errors.New("token bulunamadı")
	}
	return owner, nil
}

// Transfer
func (n *QC721Token) Transfer(from, to string, tokenID uint64) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	currentOwner, exists := n.Owners[tokenID]
	if !exists {
		return errors.New("token yok")
	}
	if currentOwner != from {
		return errors.New("transfer yetkisi yok")
	}
	n.Owners[tokenID] = to
	return nil
}

// TokenURI (metadata/görsel/json)
func (n *QC721Token) TokenURI(tokenID uint64) (string, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	uri, ok := n.TokenURIs[tokenID]
	if !ok {
		return "", errors.New("token bulunamadı")
	}
	return uri, nil
}

// Ekstra: NFT'yi yak
func (n *QC721Token) Burn(tokenID uint64, owner string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	currentOwner, exists := n.Owners[tokenID]
	if !exists {
		return errors.New("token yok")
	}
	if currentOwner != owner {
		return errors.New("yakma yetkisi yok")
	}
	delete(n.Owners, tokenID)
	delete(n.TokenURIs, tokenID)
	return nil
}
