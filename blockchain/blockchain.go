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

	"quantumcoin/wallet"
)

// ---- Blockchain ----
type Blockchain struct {
	Blocks           []*Block
	UTXO             map[string][]TransactionOutput
	TotalSupply      int
	coinbaseMaturity int
	pendingTxs       []*Transaction // mempool
}

// ---- Ödül/halving ----
const (
	GenesisTime     = 1725158400              // 2024-09-01 UTC
	HalvingInterval = 2 * 365 * 24 * 60 * 60  // 2 yıl
	MiningPeriod    = 10 * 365 * 24 * 60 * 60 // 10 yıl
	InitialReward   = 50
)

func GetCurrentReward() int {
	now := time.Now().Unix()
	elapsed := now - GenesisTime
	if elapsed < 0 {
		elapsed = 0
	}
	halvings := int(elapsed / HalvingInterval)
	r := InitialReward >> halvings
	if r < 1 {
		r = 1
	}
	if elapsed > MiningPeriod {
		r = 0
	}
	return r
}

// ---- Oluşturma (GENESIS + %12 premine opsiyonel) ----
func NewBlockchain(initialReward, totalSupply int) *Blockchain {
	// 1) Genesis işlemleri
	var txs []*Transaction

	// 1.a) Basit genesis coinbase (gösterim amaçlı)
	genesisCoinbase := &Transaction{
		ID:        nil,
		Inputs:    []TransactionInput{},
		Outputs:   []TransactionOutput{{Amount: initialReward, PubKeyHash: []byte("genesis-recipient")}},
		Timestamp: time.Now(),
		Sender:    "COINBASE",
		Amount:    float64(initialReward),
	}
	genesisCoinbase.ID = genesisCoinbase.Hash()
	txs = append(txs, genesisCoinbase)

	// 1.b) %12 premine (QC_MAIN_ADDRESS env değişkeni ile)
	if mainAddr := stringsTrim(os.Getenv("QC_MAIN_ADDRESS")); mainAddr != "" && totalSupply > 0 {
		premine := int(float64(totalSupply) * 0.12) // toplam arzın %12’si
		if premine > 0 {
			txs = append(txs, &Transaction{
				ID:        nil,
				Inputs:    []TransactionInput{},
				Outputs:   []TransactionOutput{{Amount: premine, PubKeyHash: wallet.Base58DecodeAddress(mainAddr)}},
				Timestamp: time.Now(),
				Sender:    "COINBASE",
				Amount:    float64(premine),
			})
			txs[len(txs)-1].ID = txs[len(txs)-1].Hash()
		}
	}

	// 2) Genesis bloğunu PoW ile üret
	genesis := NewBlock(0, txs, []byte{}, "genesis", 1)

	bc := &Blockchain{
		Blocks:      []*Block{genesis},
		UTXO:        map[string][]TransactionOutput{},
		TotalSupply: totalSupply,
		pendingTxs:  []*Transaction{},
	}
	bc.UpdateUTXOSet()
	return bc
}

func stringsTrim(s string) string {
	// küçük yardımcı: boşlukları temizle
	return strings.TrimSpace(s)
}

// ---- Olgunlaşma ----
func (bc *Blockchain) SetCoinbaseMaturity(n int) {
	if n < 0 {
		n = 0
	}
	bc.coinbaseMaturity = n
}

// ---- Zincire blok ekleme (elle) ----
func (bc *Blockchain) AddBlock(txs []*Transaction, miner string, difficulty int) *Block {
	prev := bc.Blocks[len(bc.Blocks)-1]
	nb := NewBlock(prev.Index+1, txs, prev.Hash, miner, difficulty)
	bc.Blocks = append(bc.Blocks, nb)
	bc.UpdateUTXOSet()
	return nb
}

// ---- P2P yardımcıları ----
func (bc *Blockchain) AddBlockFromPeer(blk *Block) error {
	if !blk.ValidatePoW() {
		return fmt.Errorf("invalid proof-of-work")
	}
	if len(bc.Blocks) > 0 {
		last := bc.Blocks[len(bc.Blocks)-1]
		if !bytes.Equal(blk.PrevHash, last.Hash) {
			return fmt.Errorf("prev hash mismatch")
		}
	}
	bc.Blocks = append(bc.Blocks, blk)
	bc.UpdateUTXOSet()
	return nil
}

func (bc *Blockchain) IsValidChain() bool {
	for i := 1; i < len(bc.Blocks); i++ {
		if !bc.Blocks[i].ValidatePoW() || !bytes.Equal(bc.Blocks[i].PrevHash, bc.Blocks[i-1].Hash) {
			return false
		}
	}
	return true
}

func (bc *Blockchain) GetHeight() int { return len(bc.Blocks) - 1 }

func (bc *Blockchain) ReplaceChain(blocks []*Block) error {
	if len(blocks) <= len(bc.Blocks) {
		return fmt.Errorf("incoming chain is not longer")
	}
	for i := 1; i < len(blocks); i++ {
		if !blocks[i].ValidatePoW() || !bytes.Equal(blocks[i].PrevHash, blocks[i-1].Hash) {
			return fmt.Errorf("incoming chain is invalid")
		}
	}
	bc.Blocks = blocks
	bc.UpdateUTXOSet()
	return nil
}

func (bc *Blockchain) GetAllBlocks() []*Block { return bc.Blocks }

// ---- UTXO ----
func (bc *Blockchain) FindSpendableOutputs(pubKeyHash []byte, amount int) (map[string][]int, int) {
	acc := 0
	unspent := make(map[string][]int)
	for txID, outs := range bc.UTXO {
		for idx, out := range outs {
			if out.IsLockedWithKey(pubKeyHash) {
				acc += out.Amount
				unspent[txID] = append(unspent[txID], idx)
				if acc >= amount {
					return unspent, acc
				}
			}
		}
	}
	return unspent, acc
}

func (bc *Blockchain) UpdateUTXOSet() {
	UTXO := make(map[string][]TransactionOutput)
	for _, block := range bc.Blocks {
		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)
			for outIdx, out := range tx.Outputs {
				spent := false
				for _, ob := range bc.Blocks {
					if spent {
						break
					}
					for _, otx := range ob.Transactions {
						if spent {
							break
						}
						for _, in := range otx.Inputs {
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

// ---- Mempool ----
func (bc *Blockchain) AddTransaction(tx *Transaction) error {
	if tx == nil {
		return fmt.Errorf("nil transaction")
	}
	bc.pendingTxs = append(bc.pendingTxs, tx)
	return nil
}

// ---- Bakiye ----
func (bc *Blockchain) GetSpendableBalance(address string) int {
	pubKeyHash := wallet.Base58DecodeAddress(address)
	best := bc.GetBestHeight()
	spend := 0

	for height, block := range bc.Blocks {
		for _, tx := range block.Transactions {
			for outIdx, out := range tx.Outputs {
				if !out.IsLockedWithKey(pubKeyHash) || bc.isOutputSpent(tx.ID, outIdx) {
					continue
				}
				if tx.IsCoinbase() && bc.coinbaseMaturity > 0 {
					age := best - height
					if age < bc.coinbaseMaturity {
						continue
					}
				}
				spend += out.Amount
			}
		}
	}
	return spend
}

func (bc *Blockchain) GetBalance(address string) int {
	pubKeyHash := wallet.Base58DecodeAddress(address)
	total := 0
	for _, outs := range bc.UTXO {
		for _, out := range outs {
			if out.IsLockedWithKey(pubKeyHash) {
				total += out.Amount
			}
		}
	}
	return total
}

func (bc *Blockchain) TotalMinted() int {
	total := 0
	for _, b := range bc.Blocks {
		for _, tx := range b.Transactions {
			if tx.IsCoinbase() {
				for _, out := range tx.Outputs {
					total += out.Amount
				}
			}
		}
	}
	return total
}

// ---- MineBlock: coinbase + mempool ----
func (bc *Blockchain) MineBlock(miner string, difficulty int) (*Block, error) {
	reward := GetCurrentReward()
	if reward == 0 {
		return nil, fmt.Errorf("madencilik dönemi sona erdi")
	}
	if bc.TotalSupply > 0 {
		rem := bc.TotalSupply - bc.TotalMinted()
		if rem <= 0 {
			return nil, fmt.Errorf("toplam arz tükendi")
		}
		if reward > rem {
			reward = rem
		}
	}
	cb := &Transaction{
		ID:        nil,
		Inputs:    []TransactionInput{},
		Outputs:   []TransactionOutput{{Amount: reward, PubKeyHash: wallet.Base58DecodeAddress(miner)}},
		Timestamp: time.Now(),
		Sender:    "COINBASE",
		Amount:    float64(reward),
	}
	cb.ID = cb.Hash()

	txs := append([]*Transaction{cb}, bc.pendingTxs...)
	prev := bc.Blocks[len(bc.Blocks)-1]
	nb := NewBlock(prev.Index+1, txs, prev.Hash, miner, difficulty)

	bc.Blocks = append(bc.Blocks, nb)
	bc.UpdateUTXOSet()
	bc.pendingTxs = []*Transaction{} // mempool boşalt

	return nb, nil
}

// ---- Serialize/Load ----
func SerializeBlockchain(bc *Blockchain) []byte {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(bc); err != nil {
		log.Panicf("serialize error: %v", err)
	}
	return buf.Bytes()
}

func DeserializeBlockchain(data []byte) *Blockchain {
	var bc Blockchain
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&bc); err != nil {
		log.Panicf("deserialize error: %v", err)
	}
	if bc.pendingTxs == nil {
		bc.pendingTxs = []*Transaction{}
	}
	return &bc
}

func (bc *Blockchain) SaveToFile(filename string) error {
	return os.WriteFile(filename, SerializeBlockchain(bc), 0644)
}

func LoadBlockchainFromFile(filename string) (*Blockchain, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return DeserializeBlockchain(data), nil
}

// ---- helpers ----
func (bc *Blockchain) GetBestHeight() int {
	if bc == nil || len(bc.Blocks) == 0 {
		return -1
	}
	return bc.Blocks[len(bc.Blocks)-1].Index
}

func (bc *Blockchain) GetLastBlock() *Block {
	if bc == nil || len(bc.Blocks) == 0 {
		return nil
	}
	return bc.Blocks[len(bc.Blocks)-1]
}

func (bc *Blockchain) GetBlockByIndex(idx int) *Block {
	for _, b := range bc.Blocks {
		if b.Index == idx {
			return b
		}
	}
	return nil
}

func (bc *Blockchain) GetBlockByHash(hash []byte) *Block {
	for _, b := range bc.Blocks {
		if bytes.Equal(b.Hash, hash) {
			return b
		}
	}
	return nil
}

func (bc *Blockchain) isOutputSpent(txid []byte, outIdx int) bool {
	for _, blk := range bc.Blocks {
		for _, tx := range blk.Transactions {
			for _, in := range tx.Inputs {
				if bytes.Equal(in.TxID, txid) && in.OutIndex == outIdx {
					return true
				}
			}
		}
	}
	return false
}
