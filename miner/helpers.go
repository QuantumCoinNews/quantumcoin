package miner

import (
	"fmt"
	"log"

	"quantumcoin/blockchain"
)

// Bu dosya YARDIMCI işlevler içindir.
// ÖNEMLİ: MiningStatus tipi ve LogBlock fonksiyonu miner.go içindedir.
// Burada aynı isimleri TEKRAR TANIMLAmıyoruz ki derleyici çakışma vermesin.

// LogBlockVerbose: ayrıntılı log isteyen eski çağrılar için nazik sarmalayıcı.
// Yeni kodda doğrudan miner.LogBlock(b) kullanabilirsiniz.
func LogBlockVerbose(b *blockchain.Block) {
	if b == nil {
		return
	}
	LogBlock(b) // miner.go'daki asıl fonksiyon
	log.Printf("ℹ️  (verbose) block index=%d prev=%x txs=%d", b.Index, b.PrevHash, len(b.Transactions))
}

// FormatStatus: MiningStatus'i tek satır stringe çevirir (UI/log kolaylığı).
func FormatStatus(st MiningStatus) string {
	return fmt.Sprintf("height=%d time=%s hash=%x reward=%d",
		st.BlockHeight, st.Timestamp.Format("2006-01-02 15:04:05"),
		st.BlockHash, st.Reward,
	)
}
