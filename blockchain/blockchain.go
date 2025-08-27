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

type Blockchain struct {
	Blocks           []*Block
	UTXO             map[string][]TransactionOutput
	TotalSupply      int
	coinbaseMaturity int
	pendingTxs       []*Transaction
}

// Varsayılanlar (config yoksa devreye girer)
const (
	GenesisTimeDefault     = 1725158400             // 2024-09-01 UTC
	HalvingIntervalDefault = 2 * 365 * 24 * 60 * 60 // ~2 yıl
	MiningPeriodDefault    = 10 * 365 * 24 * 60 * 60
	InitialRewardDefault   = 50
)

func GetCurrentReward() int {
	p := config.Current()
	now := time.Now().Unix()

	genesis := p.GenesisUnix
	if genesis <= 0 {
		genesis = GenesisTimeDefault
	}
	elapsed := now - genesis
	if elapsed < 0 {
		elapsed = 0
	}

	halvingSecs := p.HalvingIntervalSecs
	if halvingSecs <= 0 {
		halvingSecs = HalvingIntervalDefault
	}
	halvings := int(elapsed / halvingSecs)

	r := p.InitialReward
	if r <= 0 {
		r = InitialRewardDefault
	}
	r >>= halvings
	if r < 1 {
		r = 1
	}
	miningSecs := p.MiningPeriodSecs
	if miningSecs <= 0 {
		miningSecs = MiningPeriodDefault
	}
	if elapsed > miningSecs {
		r = 0
	}
	return r
}

func NewBlockchain(initialReward, totalSupply int) *Blockchain {
	var txs []*Transaction

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

	if mainAddr := strings.TrimSpace(os.Getenv("QC_MAIN_ADDRESS")); mainAddr != "" && totalSupply > 0 {
		premine := int(float64(totalSupply) * 0.12)
		if premine > 0 {
			pk := wallet.Base58DecodeAddress(mainAddr)
			premTx := &Transaction{
				ID:        nil,
				Inputs:    []TransactionInput{},
				Outputs:   []TransactionOutput{{Amount: premine, PubKeyHash: pk}},
				Timestamp: time.Now(),
				Sender:    "COINBASE",
				Amount:    float64(premine),
			}
			premTx.ID = premTx.Hash()
			txs = append(txs, premTx)
		}
	}

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

func (bc *Blockchain) SetCoinbaseMaturity(n int) {
	if n < 0 {
		n = 0
	}
	bc.coinbaseMaturity = n
}

func (bc *Blockchain) AddBlock(txs []*Transaction, miner string, difficulty int) *Block {
	// Blok içindeki işlemleri doğrula (coinbase hariç imza zorunlu)
	if err := bc.validateBlockTxs(txs); err != nil {
		log.Printf("rejecting block: %v", err)
		return nil
	}

	prev := bc.Blocks[len(bc.Blocks)-1]
	nb := NewBlock(prev.Index+1, txs, prev.Hash, miner, difficulty)
	bc.Blocks = append(bc.Blocks, nb)
	bc.UpdateUTXOSet()
	return nb
}

func (bc *Blockchain) AddBlockFromPeer(blk *Block) error {
	if !blk.ValidatePoW() {
		return ErrInvalidPoW
	}
	if len(bc.Blocks) > 0 {
		last := bc.Blocks[len(bc.Blocks)-1]
		if !bytes.Equal(blk.PrevHash, last.Hash) {
			return ErrPrevHashMismatch
		}
	}
	// Peer'den gelen bloğun işlemlerini doğrula
	if err := bc.validateBlockTxs(blk.Transactions); err != nil {
		return err
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
		return ErrIncomingChainNotLonger
	}
	for i := 1; i < len(blocks); i++ {
		if !blocks[i].ValidatePoW() || !bytes.Equal(blocks[i].PrevHash, blocks[i-1].Hash) {
			return ErrIncomingChainInvalid
		}
		// Zincir değiştirmede her bloğun işlemlerini denetle
		if err := bc.validateBlockTxs(blocks[i].Transactions); err != nil {
			return fmt.Errorf("incoming chain invalid tx: %w", err)
		}
	}
	bc.Blocks = blocks
	bc.UpdateUTXOSet()
	return nil
}

func (bc *Blockchain) GetAllBlocks() []*Block { return bc.Blocks }

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
	utxo := make(map[string][]TransactionOutput)
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
					utxo[txID] = append(utxo[txID], out)
				}
			}
		}
	}
	bc.UTXO = utxo
}

// --- İMZA ZORUNLULUĞU: mempool’a eklemeden önce doğrula ---
func (bc *Blockchain) AddTransaction(tx *Transaction) error {
	if tx == nil {
		return ErrNilTransaction
	}
	// coinbase dışındaki işlemler imzalı ve doğrulanmış olmalı
	if !tx.IsCoinbase() && !tx.Verify() {
		return fmt.Errorf("invalid tx signature")
	}
	// basit kurallar
	if len(tx.Outputs) == 0 {
		return fmt.Errorf("empty outputs")
	}
	if !tx.IsCoinbase() && len(tx.Inputs) == 0 {
		return fmt.Errorf("empty inputs")
	}

	bc.pendingTxs = append(bc.pendingTxs, tx)
	return nil
}

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

func (bc *Blockchain) MineBlock(miner string, difficulty int) (*Block, error) {
	if len(bc.Blocks) == 0 {
		return nil, ErrChainNotInitialized
	}
	cbTx, err := newCoinbaseTx(miner)
	if err != nil {
		return nil, fmt.Errorf("coinbase tx: %w", err) // wrapcheck
	}

	// pending kopyasını al, coinbase ile birleştir ve doğrula
	txs := append([]*Transaction{cbTx}, bc.PendingTxs()...)
	if err := bc.validateBlockTxs(txs); err != nil {
		return nil, err
	}

	prev := bc.Blocks[len(bc.Blocks)-1]
	nb := NewBlock(prev.Index+1, txs, prev.Hash, miner, difficulty)

	bc.Blocks = append(bc.Blocks, nb)
	bc.UpdateUTXOSet()
	bc.pendingTxs = []*Transaction{} // mempool’u boşalt

	return nb, nil
}

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
	return os.WriteFile(filename, SerializeBlockchain(bc), 0o600)
}

func LoadBlockchainFromFile(filename string) (*Blockchain, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read blockchain file: %w", err) // wrapcheck
	}
	return DeserializeBlockchain(data), nil
}

// Helpers

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

// ---- Eklenen yardımcılar ----

// Blok içindeki tüm işlemleri doğrula (coinbase için sadece output kuralı)
func (bc *Blockchain) validateBlockTxs(txs []*Transaction) error {
	for _, tx := range txs {
		if tx == nil {
			return fmt.Errorf("nil tx")
		}
		if tx.IsCoinbase() {
			if len(tx.Outputs) == 0 {
				return fmt.Errorf("invalid coinbase (no outputs)")
			}
			continue
		}
		if !tx.Verify() {
			return fmt.Errorf("invalid tx signature")
		}
	}
	return nil
}

// pendingTxs'in güvenli kopyası (API/mine kullanımı için)
func (bc *Blockchain) PendingTxs() []*Transaction {
	if bc == nil || bc.pendingTxs == nil {
		return []*Transaction{}
	}
	cp := make([]*Transaction, len(bc.pendingTxs))
	copy(cp, bc.pendingTxs)
	return cp
}
