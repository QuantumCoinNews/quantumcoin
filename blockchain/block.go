package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"time"
)

// Block: QuantumCoin zincirinin temel yapı taşı
type Block struct {
	Index        int
	Timestamp    int64
	Transactions []*Transaction
	PrevHash     []byte
	Hash         []byte
	Nonce        int
	Miner        string
	Difficulty   int
	// Genişletilebilir alan: NFT, bonus, metadata vs eklenebilir
	// NFTID      string
	// Reward     int
}

// NewBlock: Zincire yeni blok ekler
func NewBlock(index int, txs []*Transaction, prevHash []byte, miner string, difficulty int) *Block {
	block := &Block{
		Index:        index,
		Timestamp:    time.Now().Unix(),
		Transactions: txs,
		PrevHash:     prevHash,
		Hash:         []byte{},
		Nonce:        0,
		Miner:        miner,
		Difficulty:   difficulty,
	}
	pow := NewProofOfWork(block) // pow.go içinde olmalı
	nonce, hash := pow.Run()
	block.Nonce = nonce
	block.Hash = hash
	return block
}

// HashTransactions: İşlemleri tek hash'e dönüştürür
func (b *Block) HashTransactions() []byte {
	var txHashes [][]byte
	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.Hash()) // transaction.go'da Hash() fonksiyonu gerekli!
	}
	data := bytes.Join(txHashes, []byte{})
	hash := sha256.Sum256(data)
	return hash[:]
}

// Serialize: Blok'u []byte olarak kodlar (gob)
func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(b)
	if err != nil {
		panic(err)
	}
	return result.Bytes()
}

// DeserializeBlock: []byte'dan Block objesi üretir
func DeserializeBlock(data []byte) *Block {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&block)
	if err != nil {
		panic(err)
	}
	return &block
}

// CreateGenesisBlock: İlk bloğu zincire yazar
func CreateGenesisBlock(reward int) *Block {
	genesisTx := &Transaction{
		ID:     nil,
		Inputs: []TransactionInput{},
		Outputs: []TransactionOutput{
			{
				Amount:     reward,
				PubKeyHash: []byte("genesis-recipient"), // İleride gerçek cüzdan adresi!
			},
		},
	}
	genesisTx.ID = genesisTx.Hash()
	return NewBlock(0, []*Transaction{genesisTx}, []byte{}, "genesis", 1)
}
