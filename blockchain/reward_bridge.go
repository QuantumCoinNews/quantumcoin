package blockchain

import (
	"crypto/sha256"
	"encoding/binary"
	"time"
)

// Basit “bridge” — dış bağımlılık yok.
// Varsayım: 1 QC = 1 “atom” (ölçekleme yok).

// nftDropEligible: ~%1 olasılıkla NFT ödülü
func nftDropEligible(height int64, blockHash []byte) (bool, string) {
	h := sha256.Sum256(append(blockHash,
		byte(height), byte(height>>8), byte(height>>16), byte(height>>24)))
	v := binary.BigEndian.Uint16(h[0:2]) // 0..65535
	ok := v < 655                        // ~%1
	h2 := sha256.Sum256(h[:])
	return ok, hexString(h2[:])
}

func hexString(b []byte) string {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, c := range b {
		out[i*2] = hexdigits[c>>4]
		out[i*2+1] = hexdigits[c&0x0f]
	}
	return string(out)
}

// ComputeRewardBridge: temel dağıtım (tamamı madenciye)
func ComputeRewardBridge(
	height int64,
	now int64,
	totalMintedQC int64,
	blockHash []byte,
	fees int64,
	_ /* cfg placeholder */ any,
) RewardBreakdown {
	if now == 0 {
		now = time.Now().Unix()
	}
	base := int64(GetCurrentReward()) // QC → atom 1:1

	// Tamamını madenci alsın (basit profil)
	toMiner := base + fees
	rb := RewardBreakdown{
		Height:        height,
		Timestamp:     now,
		BaseSubsidy:   base,
		AnnualBonus:   0,
		FeesCollected: fees,
		ToMiner:       toMiner,
		ToStakers:     0,
		ToDev:         0,
		ToBurn:        0,
		ToCommunity:   0,
		Total:         toMiner,
	}
	ok, det := nftDropEligible(height, blockHash)
	rb.NFTAwarded = ok
	rb.NFTDeterminism = det
	return rb
}

func (rb RewardBreakdown) MetadataKV() map[string]string { return RewardMetadataKV(rb) }
