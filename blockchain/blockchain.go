package blockchain

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"quantumcoin/config"
	"quantumcoin/wallet"
)

// ---- Blockchain veri yapısı ----

type Blockchain struct {
	Blocks           []*Block
	UTXO             map[string][]TransactionOutput
	TotalSupply      int // toplam arz sınırı; NewBlockchain parametresiyle gelir (0 = sınırsız)
	coinbaseMaturity int // harcanabilir olması için beklenmesi gereken blok sayısı (0 => hemen)
}

// ---- Sabitler: Halving ve Mining parametreleri ----

const (
	GenesisTime     = 1725158400              // 2024-09-01 00:00:00 UTC (örnek)
	HalvingInterval = 2 * 365 * 24 * 60 * 60  // 2 yıl = saniye
	MiningPeriod    = 10 * 365 * 24 * 60 * 60 // 10 yıl = saniye
	InitialReward   = 50
)

// ---- Halving’e göre blok ödülü hesapla ----

func GetCurrentReward() int {
	now := time.Now().Unix()
	elapsed := now - GenesisTime
	if elapsed < 0 {
		elapsed = 0
	}
	halvings := int(elapsed / HalvingInterval)
	reward := InitialReward >> halvings // 50 → 25 → 12 → 6 → 3 → ...
	if reward < 1 {
		reward = 1
	}
	if elapsed > MiningPeriod {
		reward = 0
	}
	return reward
}

// ---- Blockchain oluşturma ----

func NewBlockchain(initialReward int, totalSupply int) *Blockchain {
	genesis := CreateGenesisBlock(initialReward)
	bc := &Blockchain{
		Blocks:      []*Block{genesis},
		UTXO:        make(map[string][]TransactionOutput),
		TotalSupply: totalSupply,
	}
	bc.UpdateUTXOSet()
	return bc
}

// Coinbase olgunlaşma ayarı (blok sayısı).
func (bc *Blockchain) SetCoinbaseMaturity(n int) {
	if n < 0 {
		n = 0
	}
	bc.coinbaseMaturity = n
}

// ---- Blok ekleme ----

func (bc *Blockchain) AddBlock(transactions []*Transaction, miner string, difficulty int) *Block {
	prevBlock := bc.Blocks[len(bc.Blocks)-1]
	newBlock := NewBlock(prevBlock.Index+1, transactions, prevBlock.Hash, miner, difficulty)
	bc.Blocks = append(bc.Blocks, newBlock)
	bc.UpdateUTXOSet()
	return newBlock
}

// ---- UTXO & TX yönetimi ----

func (bc *Blockchain) FindSpendableOutputs(pubKeyHash []byte, amount int) (map[string][]int, int) {
	accumulated := 0
	unspentOutputs := make(map[string][]int)
	for txID, outs := range bc.UTXO {
		for idx, out := range outs {
			if out.IsLockedWithKey(pubKeyHash) {
				accumulated += out.Amount
				unspentOutputs[txID] = append(unspentOutputs[txID], idx)
				if accumulated >= amount {
					return unspentOutputs, accumulated
				}
			}
		}
	}
	return unspentOutputs, accumulated
}

func (bc *Blockchain) UpdateUTXOSet() {
	UTXO := make(map[string][]TransactionOutput)
	for _, block := range bc.Blocks {
		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)
			for outIdx, out := range tx.Outputs {
				spent := false
				// Basit tarama: başka işlemlerde bu çıkış harcanmış mı?
				for _, otherBlock := range bc.Blocks {
					if spent {
						break
					}
					for _, otherTx := range otherBlock.Transactions {
						if spent {
							break
						}
						for _, in := range otherTx.Inputs {
							if hex.EncodeToString(in.TxID) == txID && in.OutIndex == outIdx {
								spent = true
								break
							}
						}
					}
				}
				if !spent {
					UTXO[txID] = append(UTXO[txID], out)
				}
			}
		}
	}
	bc.UTXO = UTXO
}

func (bc *Blockchain) AddTransaction(tx *Transaction) error {
	if !tx.Verify() {
		return fmt.Errorf("geçersiz işlem")
	}
	// İleride: mempool'a ekle
	return nil
}

// ---- Yardımcı: belirli bir çıkış harcanmış mı? ----

func (bc *Blockchain) isOutputSpent(txID []byte, outIdx int) bool {
	for _, b := range bc.Blocks {
		for _, t := range b.Transactions {
			for _, in := range t.Inputs {
				if bytes.Equal(in.TxID, txID) && in.OutIndex == outIdx {
					return true
				}
			}
		}
	}
	return false
}

// ---- Spendable Balance (coinbase maturity ile) ----

func (bc *Blockchain) GetSpendableBalance(address string) int {
	pubKeyHash := wallet.Base58DecodeAddress(address)
	best := bc.GetBestHeight()
	spendable := 0

	for height, block := range bc.Blocks {
		for _, tx := range block.Transactions {
			for outIdx, out := range tx.Outputs {
				if !out.IsLockedWithKey(pubKeyHash) {
					continue
				}
				// Zaten harcanmış mı?
				if bc.isOutputSpent(tx.ID, outIdx) {
					continue
				}
				// Coinbase ise olgunlaşma kontrolü
				if tx.IsCoinbase() && bc.coinbaseMaturity > 0 {
					age := best - height
					if age < bc.coinbaseMaturity {
						continue // henüz harcanamaz
					}
				}
				spendable += out.Amount
			}
		}
	}
	return spendable
}

// ---- Toplam basılan ödül hesapla ----

func (bc *Blockchain) TotalMinted() int {
	total := 0
	for _, block := range bc.Blocks {
		for _, tx := range block.Transactions {
			if tx.IsCoinbase() {
				for _, out := range tx.Outputs {
					total += out.Amount
				}
			}
		}
	}
	return total
}

// ---- Madencilik (HALVING + ARZ SINIRI + COINBASE SPLIT) ----

func (bc *Blockchain) MineBlock(miner string, difficulty int) (*Block, error) {
	// Halving’e göre baz ödül
	reward := GetCurrentReward()
	if reward == 0 {
		return nil, fmt.Errorf("madencilik dönemi sona erdi")
	}

	// Toplam arz sınırı uygulanır (parametre verilmemişse 0 => sınırsız)
	if bc.TotalSupply > 0 {
		remaining := bc.TotalSupply - bc.TotalMinted()
		if remaining <= 0 {
			return nil, fmt.Errorf("toplam arz tükendi")
		}
		if reward > remaining {
			reward = remaining
		}
	}

	// === COINBASE SPLIT ===
	cfg := config.Current()
	outs := make([]TransactionOutput, 0, 5)

	// yüzdeler
	pctMiner := cfg.RewardPctMiner
	pctStake := cfg.RewardPctStake
	pctDev := cfg.RewardPctDev
	pctBurn := cfg.RewardPctBurn
	allocated := pctMiner + pctStake + pctDev + pctBurn
	pctComm := 0
	if allocated < 100 {
		pctComm = 100 - allocated
	}

	part := func(p int) int {
		if p <= 0 {
			return 0
		}
		return (reward * p) / 100
	}
	minerAmt := part(pctMiner)
	stakeAmt := part(pctStake)
	devAmt := part(pctDev)
	burnAmt := part(pctBurn)
	commAmt := part(pctComm)

	// toplam yuvarlama farkını miner'a ekle
	sum := minerAmt + stakeAmt + devAmt + burnAmt + commAmt
	if diff := reward - sum; diff != 0 {
		minerAmt += diff
	}

	// hedef adres helper
	addOut := func(addr string, amt int) {
		if amt <= 0 || strings.TrimSpace(addr) == "" {
			return
		}
		outs = append(outs, TransactionOutput{
			Amount:     amt,
			PubKeyHash: wallet.Base58DecodeAddress(addr),
		})
	}

	// Miner adresi: cfg.RewardAddrMiner boşsa parametreyi kullan
	minerAddr := cfg.RewardAddrMiner
	if strings.TrimSpace(minerAddr) == "" {
		minerAddr = miner
	}
	addOut(minerAddr, minerAmt)
	addOut(cfg.RewardAddrStake, stakeAmt)
	addOut(cfg.RewardAddrDev, devAmt)
	addOut(cfg.RewardAddrBurn, burnAmt)
	addOut(cfg.RewardAddrCommunity, commAmt)

	// hiçbir çıkış oluşmadıysa fallback
	if len(outs) == 0 {
		outs = append(outs, TransactionOutput{
			Amount:     reward,
			PubKeyHash: wallet.Base58DecodeAddress(minerAddr),
		})
	}

	rewardTx := &Transaction{
		ID:        nil,
		Inputs:    []TransactionInput{}, // coinbase
		Outputs:   outs,
		Timestamp: time.Now(),
		Sender:    "",
		Amount:    float64(reward),
	}
	rewardTx.ID = rewardTx.Hash()

	pendingTxs := []*Transaction{rewardTx}

	prevBlock := bc.Blocks[len(bc.Blocks)-1]
	newBlock := NewBlock(prevBlock.Index+1, pendingTxs, prevBlock.Hash, miner, difficulty)
	bc.Blocks = append(bc.Blocks, newBlock)
	bc.UpdateUTXOSet()
	return newBlock, nil
}

// ---- Eski: tek çıkışlı coinbase (uyumluluk için dursun) ----

func CreateRewardTx(miner string, amount int) *Transaction {
	output := TransactionOutput{
		Amount:     amount,
		PubKeyHash: wallet.Base58DecodeAddress(miner),
	}
	tx := &Transaction{
		ID:      nil,
		Inputs:  []TransactionInput{},
		Outputs: []TransactionOutput{output},
	}
	tx.ID = tx.Hash()
	return tx
}

// ---- Serialization / Deserialization ----

func SerializeBlockchain(bc *Blockchain) []byte {
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	if err := enc.Encode(bc); err != nil {
		log.Panicf("Blockchain serializasyon hatası: %v", err)
	}
	return buffer.Bytes()
}

func DeserializeBlockchain(data []byte) *Blockchain {
	var bc Blockchain
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&bc); err != nil {
		log.Panicf("Blockchain deserialization hatası: %v", err)
	}
	return &bc
}

func (bc *Blockchain) SaveToFile(filename string) error {
	data := SerializeBlockchain(bc)
	return os.WriteFile(filename, data, 0644)
}

func LoadBlockchainFromFile(filename string) (*Blockchain, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	bc := DeserializeBlockchain(data)
	return bc, nil
}

// ---- Ekstra fonksiyonlar ----

func (bc *Blockchain) GetBestHeight() int {
	if len(bc.Blocks) == 0 {
		return -1
	}
	return bc.Blocks[len(bc.Blocks)-1].Index
}

func (bc *Blockchain) GetBalance(address string) int {
	pubKeyHash := wallet.Base58DecodeAddress(address)
	balance := 0
	for _, outs := range bc.UTXO {
		for _, out := range outs {
			if out.IsLockedWithKey(pubKeyHash) {
				balance += out.Amount
			}
		}
	}
	return balance
}

func (bc *Blockchain) GetAllBlocks() []*Block {
	return bc.Blocks
}

// GetBlockByIndex: Belirli bir index’teki bloku döndürür
func (bc *Blockchain) GetBlockByIndex(index int) *Block {
	for _, block := range bc.Blocks {
		if block.Index == index {
			return block
		}
	}
	return nil
}

// GetBlockByHash: Belirli bir hash’e sahip bloku döndürür
func (bc *Blockchain) GetBlockByHash(hash []byte) *Block {
	for _, block := range bc.Blocks {
		if bytes.Equal(block.Hash, hash) {
			return block
		}
	}
	return nil
}

// GetLastBlock: Son bloku döndürür
func (bc *Blockchain) GetLastBlock() *Block {
	if len(bc.Blocks) == 0 {
		return nil
	}
	return bc.Blocks[len(bc.Blocks)-1]
}

// FindTransaction: Belirli TxID ile işlemi bulur
func (bc *Blockchain) FindTransaction(ID []byte) (*Transaction, error) {
	for _, block := range bc.Blocks {
		for _, tx := range block.Transactions {
			if bytes.Equal(tx.ID, ID) {
				return tx, nil
			}
		}
	}
	return nil, fmt.Errorf("transaction not found")
}

// ---- Zincir doğrulama & değiştirme & peer blok ekleme ----

func (bc *Blockchain) GetHeight() int { // p2p tarafı kullandıysa
	return bc.GetBestHeight()
}

// Basit zincir geçerlilik kontrolü: prev hash, PoW, index artışı
func (bc *Blockchain) IsValidChain() bool {
	if len(bc.Blocks) == 0 {
		return false
	}
	for i := 1; i < len(bc.Blocks); i++ {
		prev := bc.Blocks[i-1]
		cur := bc.Blocks[i]
		if !bytes.Equal(cur.PrevHash, prev.Hash) {
			return false
		}
		if cur.Index != prev.Index+1 {
			return false
		}
		pow := NewProofOfWork(cur)
		if !pow.Validate() {
			return false
		}
	}
	return true
}

// ReplaceChain: daha uzun ve geçerli zincir ile değiştir
func (bc *Blockchain) ReplaceChain(blocks []*Block) error {
	if len(blocks) <= len(bc.Blocks) {
		return fmt.Errorf("incoming chain not longer")
	}
	// geçerlilik kontrolü
	for i := 1; i < len(blocks); i++ {
		if !bytes.Equal(blocks[i].PrevHash, blocks[i-1].Hash) {
			return fmt.Errorf("incoming chain prevhash mismatch at %d", i)
		}
		pow := NewProofOfWork(blocks[i])
		if !pow.Validate() {
			return fmt.Errorf("incoming chain pow invalid at %d", i)
		}
	}
	bc.Blocks = blocks
	bc.UpdateUTXOSet()
	return nil
}

// AddBlockFromPeer: dışardan gelen blok ekle (kurallara uygun ise)
func (bc *Blockchain) AddBlockFromPeer(b *Block) error {
	if len(bc.Blocks) == 0 {
		return fmt.Errorf("local chain empty")
	}
	tip := bc.Blocks[len(bc.Blocks)-1]

	if !bytes.Equal(b.PrevHash, tip.Hash) {
		return fmt.Errorf("prev hash mismatch")
	}
	if b.Index != tip.Index+1 {
		return fmt.Errorf("index mismatch")
	}
	pow := NewProofOfWork(b)
	if !pow.Validate() {
		return fmt.Errorf("invalid pow")
	}

	// (Opsiyonel) coinbase/tx doğrulamaları burada genişletilebilir

	bc.Blocks = append(bc.Blocks, b)
	bc.UpdateUTXOSet()
	return nil
}
