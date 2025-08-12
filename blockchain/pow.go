package blockchain

import (
	"bytes"
	"crypto/sha256"
	"math"
	"math/big"
	"strconv"
)

const maxNonce = math.MaxInt64
const defaultDifficultyBits = 16 // config default ile uyumlu

// ProofOfWork: QuantumCoin PoW algoritması
type ProofOfWork struct {
	Block      *Block
	Target     *big.Int
	Difficulty int
}

// NewProofOfWork: yeni PoW objesi
func NewProofOfWork(block *Block) *ProofOfWork {
	// Zorluk 1..255 aralığına sıkıştırılır; 0 gelirse güvenli varsayılan kullan
	diff := block.Difficulty
	if diff <= 0 {
		diff = defaultDifficultyBits
	}
	if diff > 255 {
		diff = 255
	}
	target := big.NewInt(1)
	target.Lsh(target, uint(256-diff)) // Zorluk seviyesine göre hedef ayarlanır

	return &ProofOfWork{
		Block:      block,
		Target:     target,
		Difficulty: diff,
	}
}

// Run: nonce bulma işlemi (madencilik)
func (pow *ProofOfWork) Run() (int, []byte) {
	var hashInt big.Int
	var hash [32]byte
	nonce := 0

	for nonce < maxNonce {
		data := pow.prepareData(nonce)
		hash = sha256.Sum256(data)
		hashInt.SetBytes(hash[:])
		if hashInt.Cmp(pow.Target) == -1 {
			break
		}
		nonce++
	}
	return nonce, hash[:]
}

// prepareData: Blok + nonce + diğer alanlar hash’lenir
func (pow *ProofOfWork) prepareData(nonce int) []byte {
	data := bytes.Join([][]byte{
		pow.Block.PrevHash,
		pow.Block.HashTransactions(),
		[]byte(strconv.Itoa(pow.Block.Index)),
		[]byte(strconv.FormatInt(pow.Block.Timestamp, 10)), // (YENİ) timestamp dahil
		[]byte(strconv.Itoa(nonce)),
		[]byte(strconv.Itoa(pow.Difficulty)),
		[]byte(pow.Block.Miner),
		// Genişletme alanı: NFT/metadata eklenecekse buraya!
	}, []byte{})
	return data
}

// Validate: Blok hash’i PoW koşulunu sağlıyor mu?
func (pow *ProofOfWork) Validate() bool {
	var hashInt big.Int
	data := pow.prepareData(pow.Block.Nonce)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])
	return hashInt.Cmp(pow.Target) == -1
}
