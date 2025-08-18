// blockchain/nft.go
package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// MintNFT: Basit/stub NFT mint işlemi.
// - Deterministik pseudo-ID üretir (adres + tip + zaman + yükseklik).
// - Son bloğun Metadata alanına iz bırakır ve gelen meta'yı prefiksleyerek kaydeder.
func (bc *Blockchain) MintNFT(toAddress, nftType string, meta map[string]string) (string, error) {
	if bc == nil {
		return "", fmt.Errorf("blockchain is nil")
	}
	last := bc.GetLastBlock()
	if last == nil {
		return "", fmt.Errorf("no blocks in chain")
	}

	// ID oluştur
	base := fmt.Sprintf("%s|%s|%d|%d", toAddress, nftType, time.Now().UnixNano(), len(bc.Blocks))
	sum := sha256.Sum256([]byte(base))
	nftID := hex.EncodeToString(sum[:])

	// Metadata kayıtları
	if last.Metadata == nil {
		last.Metadata = map[string]string{}
	}
	last.Metadata["last_nft_mint"] = fmt.Sprintf("%s:%s:%s", toAddress, nftType, nftID)
	if toAddress != "" {
		last.Metadata[fmt.Sprintf("nft_last_%s", toAddress)] = nftID
	}
	// Gelen meta'yı da izlemek için prefiksli kaydet
	if meta != nil {
		for k, v := range meta {
			last.Metadata["nft_meta_"+k] = v
		}
	}

	return nftID, nil
}
