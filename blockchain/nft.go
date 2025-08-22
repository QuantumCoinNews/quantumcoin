package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

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

	// Gelen meta'yı da izlemek için prefiksli kaydet (nil map üzerinde range güvenlidir)
	for k, v := range meta {
		last.Metadata["nft_meta_"+k] = v
	}

	return nftID, nil
}
