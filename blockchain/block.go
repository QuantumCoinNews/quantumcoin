// blockchain/block.go
package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"time"

	"quantumcoin/config"
	"quantumcoin/utils"
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
	Metadata     map[string]string // Genişletilebilir metadata
}

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
		Metadata:     map[string]string{},
	}
	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()
	block.Nonce = nonce
	block.Hash = hash
	return block
}

func (b *Block) HashTransactions() []byte {
	var txHashes [][]byte
	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.Hash())
	}
	data := bytes.Join(txHashes, []byte{})
	sum := sha256.Sum256(data)
	return sum[:]
}

func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	enc := gob.NewEncoder(&result)
	if err := enc.Encode(b); err != nil {
		panic(err)
	}
	return result.Bytes()
}

func DeserializeBlock(data []byte) *Block {
	var block Block
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&block); err != nil {
		panic(err)
	}
	if block.Metadata == nil {
		block.Metadata = map[string]string{}
	}
	return &block
}

// Base58Check QC adresinden pubKeyHash çıkar (wallet paketine bağımlı kalmadan)
func decodeAddressPKH(address string) []byte {
	decoded, err := utils.Base58Decode([]byte(address))
	if err != nil || len(decoded) < 5 {
		return []byte{} // geçersizse boş dön
	}
	// decoded: [version][pubKeyHash][checksum]
	return decoded[1 : len(decoded)-4]
}

// CreateGenesisBlock: İlk bloğu zincire yazar (+ premine)
func CreateGenesisBlock(reward int) *Block {
	cfg := config.Current()

	genesisTx := &Transaction{
		ID:      nil,
		Inputs:  []TransactionInput{},
		Outputs: []TransactionOutput{},
	}

	// Premine: total_supply * premine_percent / 100
	premineQC := 0
	if cfg.TotalSupply > 0 && cfg.PreminePercent > 0 && cfg.PremineAddress != "" {
		premineQC = (cfg.TotalSupply * cfg.PreminePercent) / 100
	}

	if premineQC > 0 {
		pkh := decodeAddressPKH(cfg.PremineAddress)
		genesisTx.Outputs = append(genesisTx.Outputs, TransactionOutput{
			Amount:     premineQC,
			PubKeyHash: pkh,
		})
	} else {
		// Geriye dönük uyumluluk: en azından “reward” kadar dummy çıkış
		genesisTx.Outputs = append(genesisTx.Outputs, TransactionOutput{
			Amount:     reward,
			PubKeyHash: []byte("genesis-recipient"),
		})
	}

	genesisTx.ID = genesisTx.Hash()

	b := NewBlock(0, []*Transaction{genesisTx}, []byte{}, "genesis", 1)
	if b.Metadata == nil {
		b.Metadata = map[string]string{}
	}
	b.Metadata["genesis"] = "true"
	if premineQC > 0 {
		b.Metadata["premine_pct"] = fmt.Sprintf("%d", cfg.PreminePercent)
		b.Metadata["premine_addr"] = cfg.PremineAddress
		b.Metadata["premine_amount"] = fmt.Sprintf("%d", premineQC)
	}
	return b
}

func (b *Block) ValidatePoW() bool {
	pow := NewProofOfWork(b)
	return pow.Validate()
}
