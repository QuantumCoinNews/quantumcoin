package nft

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// CheckAndMintMinerNFT
// Amaç: Madenciye blok bazlı “ödül NFT” kararı verip basit bir ID döndürmek.
// Not: Bu paket zinciri bilmez; burada sadece ID üretiriz (stub).
// Zincirde metadata yazımı, blockchain.(*Blockchain).MintNFT ile yapılır.
func CheckAndMintMinerNFT(minerAddress string, blockIndex int) (string, error) {
	// Basit kural: Her 10. blokta NFT ver (stub).
	if blockIndex <= 0 || blockIndex%10 != 0 {
		return "", nil
	}
	// Deterministik kısa ID
	base := fmt.Sprintf("QC|%s|%d|%d", minerAddress, blockIndex, time.Now().UnixNano())
	sum := sha256.Sum256([]byte(base))
	id := "QC-NFT-" + hex.EncodeToString(sum[:8])
	return id, nil
}
