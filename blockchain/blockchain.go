package blockchain

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"quantumcoin/wallet"
)

// ---- Blockchain veri yapısı ----

type Blockchain struct {
	Blocks []*Block
	UTXO   map[string][]TransactionOutput
}

// ---- Sabitler: Halving ve Mining parametreleri ----

const (
	GenesisTime     = 1725158400              // Mainnet başlangıcı (ör: 2024-09-01 00:00:00 UTC)
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
		Blocks: []*Block{genesis},
		UTXO:   make(map[string][]TransactionOutput),
	}
	bc.UpdateUTXOSet()
	return bc
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
				for _, otherBlock := range bc.Blocks {
					for _, otherTx := range otherBlock.Transactions {
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
		return fmt.Errorf("Geçersiz işlem")
	}
	// İleride: mempool'a ekle
	return nil
}

// ---- Madencilik (YENİ SÜRÜM: HALVING destekli) ----

func (bc *Blockchain) MineBlock(miner string, difficulty int) (*Block, error) {
	reward := GetCurrentReward()
	if reward == 0 {
		return nil, fmt.Errorf("Madencilik dönemi sona erdi!")
	}
	rewardTx := CreateRewardTx(miner, reward)
	pendingTxs := []*Transaction{rewardTx}
	prevBlock := bc.Blocks[len(bc.Blocks)-1]
	newBlock := NewBlock(prevBlock.Index+1, pendingTxs, prevBlock.Hash, miner, difficulty)
	bc.Blocks = append(bc.Blocks, newBlock)
	bc.UpdateUTXOSet()
	return newBlock, nil
}

// ---- Ödül transferi ----

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
	err := enc.Encode(bc)
	if err != nil {
		log.Panicf("Blockchain serializasyon hatası: %v", err)
	}
	return buffer.Bytes()
}

func DeserializeBlockchain(data []byte) *Blockchain {
	var bc Blockchain
	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&bc)
	if err != nil {
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
	return nil, fmt.Errorf("Transaction not found")
}
