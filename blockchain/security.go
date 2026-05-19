// Package blockchain - security.go
//
// Bu dosya, blockchain çekirdeği için temel güvenlik / bütünlük kontrollerini içerir.
// Amaç: Mevcut yapıyı bozmadan, blok ve transaction seviyesinde "ileri seviye"
// validation katmanı eklemek.
//
// Notlar:
// - Sadece var olan tipleri (Block, Transaction, Blockchain) ve standart kütüphaneleri kullanır.
// - Ekstra bir dış bağımlılık eklemez.
// - main.go içinde QC_STRICT_CHAIN=1 ise ValidateBlockchain() çağırarak
//   tam zincir doğrulamayı aktif edebilirsin.

package blockchain

import (
	"bytes"
	"errors"
	"fmt"
	"time"
)

// --- Güvenlik limitleri (istenirse ayarlanabilir) ---

// Bir blokta izin verilen maksimum transaction sayısı.
var MaxTxPerBlock = 5000

// Bir transaction içindeki maksimum output sayısı.
var MaxOutputsPerTx = 5000

// Blok zaman damgasının "geleceğe" ne kadar taşabileceği (drift).
// Örn: max +2 saat.
var MaxFutureDrift = 2 * time.Hour

// --- Hata tipleri ---

var (
	ErrNilBlock   = errors.New("security: nil block")
	ErrNilTx      = errors.New("security: nil transaction")
	ErrEmptyChain = errors.New("security: empty blockchain")
)

// --- Transaction validation ---
//
// Transaction seviyesinde "temel" güvenlik kontrolleri.
// Kriptografik imza vb. kontrolden bağımsız, sadece yapısal doğrulama yapar.

func ValidateTransactionBasic(tx *Transaction) error {
	if tx == nil {
		return ErrNilTx
	}

	// ID boş olmasın
	if len(tx.ID) == 0 {
		return fmt.Errorf("security: tx has empty ID")
	}

	// Timestamp sıfır olmasın
	if tx.Timestamp.IsZero() {
		return fmt.Errorf("security: tx %x has zero timestamp", tx.ID)
	}

	// Output’lar
	if len(tx.Outputs) == 0 {
		// İleride "sadece veri taşıyan" tx'ler kullanacaksan bunu yumuşatabiliriz,
		// şimdilik en az 1 output bekleyelim.
		return fmt.Errorf("security: tx %x has no outputs", tx.ID)
	}

	if len(tx.Outputs) > MaxOutputsPerTx {
		return fmt.Errorf("security: tx %x has too many outputs: %d > %d",
			tx.ID, len(tx.Outputs), MaxOutputsPerTx)
	}

	// Miktarlar negatif olmasın; overflow koruması
	totalOut := 0
	for i, out := range tx.Outputs {
		if out.Amount < 0 {
			return fmt.Errorf("security: tx %x output[%d] has negative amount: %d",
				tx.ID, i, out.Amount)
		}
		// overflow koruması
		next := totalOut + out.Amount
		if next < totalOut {
			return fmt.Errorf("security: tx %x output sum overflow", tx.ID)
		}
		totalOut = next
	}

	return nil
}

// --- Block validation ---
//
// Tek bir blok için yapısal ve mantıksal temel kontroller.
// DİKKAT: Burada "geçmişe çok uzak" kontrolü YAPMIYORUZ; eski blokları
// yeniden doğrularken sorun çıkmasın diye sadece geleceğe drift kontrolü var.

func ValidateBlockBasic(b *Block) error {
	if b == nil {
		return ErrNilBlock
	}

	// Index negatif olmasın
	if b.Index < 0 {
		return fmt.Errorf("security: block has negative index: %d", b.Index)
	}

	// Hash boş olmasın
	if len(b.Hash) == 0 {
		return fmt.Errorf("security: block #%d has empty hash", b.Index)
	}

	// Genesis olmayan bloklar için PrevHash boş olmamalı
	if b.Index > 0 && len(b.PrevHash) == 0 {
		return fmt.Errorf("security: block #%d missing PrevHash", b.Index)
	}

	// Timestamp pozitif olsun
	if b.Timestamp <= 0 {
		return fmt.Errorf("security: block #%d has non-positive timestamp: %d", b.Index, b.Timestamp)
	}

	// Zaman drift kontrolü (sadece geleceğe)
	bt := time.Unix(b.Timestamp, 0)
	if bt.After(time.Now().Add(MaxFutureDrift)) {
		return fmt.Errorf("security: block #%d timestamp too far in future: %s", b.Index, bt.UTC())
	}

	// En az 1 transaction olsun (genesis blokta bile coinbase)
	if len(b.Transactions) == 0 {
		return fmt.Errorf("security: block #%d has no transactions", b.Index)
	}

	// Transaction sayısı limite göre kontrol
	if len(b.Transactions) > MaxTxPerBlock {
		return fmt.Errorf("security: block #%d has too many tx: %d > %d",
			b.Index, len(b.Transactions), MaxTxPerBlock)
	}

	// Her tx için temel validation
	for i, tx := range b.Transactions {
		if err := ValidateTransactionBasic(tx); err != nil {
			return fmt.Errorf("security: block #%d tx[%d] failed basic validation: %w", b.Index, i, err)
		}
	}

	return nil
}

// --- Chain-level validation ---
//
// Sadece struct/link tarafı, PoW ve ekonomiyi ayrıca sıkılaştırıyoruz.

func (bc *Blockchain) ValidateChainLinks() error {
	if bc == nil {
		return ErrEmptyChain
	}
	if len(bc.Blocks) == 0 {
		return ErrEmptyChain
	}

	// Genesis bloğu
	genesis := bc.Blocks[0]
	if genesis == nil {
		return fmt.Errorf("security: genesis block is nil")
	}
	if genesis.Index != 0 {
		return fmt.Errorf("security: genesis index invalid: %d (want 0)", genesis.Index)
	}
	if len(genesis.Hash) == 0 {
		return fmt.Errorf("security: genesis hash is empty")
	}

	// Genesis için temel kontrol
	if err := ValidateBlockBasic(genesis); err != nil {
		return err
	}

	// Diğer bloklar
	for i := 1; i < len(bc.Blocks); i++ {
		prev := bc.Blocks[i-1]
		cur := bc.Blocks[i]
		if cur == nil {
			return fmt.Errorf("security: block[%d] is nil", i)
		}
		// Index sıralaması
		if cur.Index != prev.Index+1 {
			return fmt.Errorf("security: bad index sequence at block[%d]: got %d, want %d",
				i, cur.Index, prev.Index+1)
		}
		// PrevHash = önceki blok hash'i
		if !bytes.Equal(cur.PrevHash, prev.Hash) {
			return fmt.Errorf("security: bad prev hash at block #%d", cur.Index)
		}

		// Blok bazında temel validation
		if err := ValidateBlockBasic(cur); err != nil {
			return err
		}
	}

	return nil
}

// ValidateBasicChain
// Zincirin link + basic block validation setini çalıştırır.
func (bc *Blockchain) ValidateBasicChain() error {
	return bc.ValidateChainLinks()
}

// ValidateFullChain
// Zincirin tümünü dolaşıp:
// - yapısal kontroller (index, prevhash, timestamp, tx sayısı)
// - PoW + prevhash (var olan IsValidChain() ile)
// - toplam arz ihlali (TotalMinted <= TotalSupply) kontrolü yapar.
func (bc *Blockchain) ValidateFullChain() error {
	if bc == nil {
		return ErrEmptyChain
	}

	// 1) Yapısal / link kontrolleri
	if err := bc.ValidateBasicChain(); err != nil {
		return err
	}

	// 2) PoW + prevhash (mevcut fonksiyonunu da devreye alalım)
	if !bc.IsValidChain() {
		return fmt.Errorf("security: pow / prevHash validation failed (IsValidChain)")
	}

	// 3) Toplam arz koruması (TotalSupply 0 ise sınırsız arz kabul ediyoruz)
	if bc.TotalSupply > 0 {
		minted := bc.TotalMinted()
		if minted > bc.TotalSupply {
			return fmt.Errorf("security: total minted %d exceeds declared supply %d", minted, bc.TotalSupply)
		}
	}

	return nil
}

// ValidateBlockchain
// Dış dünya için "tek isim": şu an ValidateFullChain()'i çağırıyoruz.
// main.go içinden QC_STRICT_CHAIN=1 ise burası devreye giriyor.
func (bc *Blockchain) ValidateBlockchain() error {
	return bc.ValidateFullChain()
}

// ValidateNewBlockBasic
// Tek bir yeni blok için, mevcut zincirin son bloğuna göre temel kontrol:
// - blok temel kuralları (ValidateBlockBasic)
// - index = lastIndex+1
// - prevHash = last.Hash
// - zaman geri gitmiyor
func (bc *Blockchain) ValidateNewBlockBasic(b *Block) error {
	if err := ValidateBlockBasic(b); err != nil {
		return err
	}

	// Zincirde blok yoksa (teorik olarak) genesis olabilir
	if len(bc.Blocks) == 0 {
		if b.Index < 0 {
			return fmt.Errorf("security: negative index for genesis: %d", b.Index)
		}
		return nil
	}

	last := bc.Blocks[len(bc.Blocks)-1]

	if b.Index != last.Index+1 {
		return fmt.Errorf("security: bad index sequence: got %d want %d", b.Index, last.Index+1)
	}

	if !bytes.Equal(b.PrevHash, last.Hash) {
		return fmt.Errorf("security: bad prev hash at height %d", b.Index)
	}

	if b.Timestamp < last.Timestamp {
		return fmt.Errorf("security: block time goes backwards at height %d", b.Index)
	}

	return nil
}

// AddBlockFromPeerSecure
// Peer'dan gelen bloklar için güvenli sarmalayıcı.
// Önce temel blok kuralları (index/prevhash/zaman/tx basic), ardından
// mevcut AddBlockFromPeer çağrılır.
func (bc *Blockchain) AddBlockFromPeerSecure(b *Block) error {
	if err := bc.ValidateNewBlockBasic(b); err != nil {
		return err
	}
	return bc.AddBlockFromPeer(b)
}
