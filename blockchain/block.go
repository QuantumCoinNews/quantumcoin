package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"time"
)

type Block struct {
	Index            int
	Timestamp        int64
	Transactions     []*Transaction
	PrevHash         []byte
	Hash             []byte
	Nonce            int
	Miner            string
	Difficulty       int
	Metadata         map[string]string
	NFTMintRecordsV1 []NFTMintRecordV1
	NFTAwardProofsV1 []NFTAwardProofV1
}

func NewBlock(
	index int,
	txs []*Transaction,
	prevHash []byte,
	miner string,
	difficulty int,
) (*Block, error) {
	b := &Block{
		Index:        index,
		Timestamp:    time.Now().Unix(),
		Transactions: txs,
		PrevHash:     prevHash,
		Hash:         nil,
		Nonce:        0,
		Miner:        miner,
		Difficulty:   difficulty,
		Metadata:     map[string]string{},
	}

	pow := NewProofOfWork(b)

	nonce, hash, err := pow.Run()
	if err != nil {
		return nil, fmt.Errorf(
			"mine block proof of work: %w",
			err,
		)
	}

	b.Nonce = nonce
	b.Hash = append([]byte(nil), hash...)

	return b, nil
}

func (b *Block) HashTransactions() []byte {
	if len(b.Transactions) == 0 {
		sum := sha256.Sum256(nil)

		return sum[:]
	}

	// prealloc: lint uyarısını sıfırlamak için kapasiteyi önceden ayarladık.
	joined := make([][]byte, 0, len(b.Transactions))
	for _, tx := range b.Transactions {
		joined = append(joined, tx.Hash())
	}

	// bytes.Join'da ikinci parametreyi nil vermek hem daha temiz hem de linter dostu.
	data := bytes.Join(joined, nil)

	sum := sha256.Sum256(data)

	return sum[:]
}

func (b *Block) Serialize() []byte {
	var buf bytes.Buffer

	// Gob encode hatası pratikte beklenmez; bilinçli olarak yok sayıyoruz.
	_ = gob.NewEncoder(&buf).Encode(b)

	return buf.Bytes()
}

func DeserializeBlock(data []byte) *Block {
	var blk Block

	_ = gob.NewDecoder(bytes.NewReader(data)).Decode(&blk)

	if blk.Metadata == nil {
		blk.Metadata = map[string]string{}
	}

	return &blk
}

func (b *Block) ValidatePoW() bool {
	return ValidateBlockPoWV1(b) == nil
}
