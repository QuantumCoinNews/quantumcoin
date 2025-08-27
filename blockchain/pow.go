package blockchain

import (
	"bytes"
	"crypto/sha256"
	"math/big"
)

const (
	hashBits = 256
	maxBits  = 255
	maxNonce = int(^uint(0) >> 1) // platform int için güvenli üst sınır
)

type ProofOfWork struct {
	block  *Block
	target *big.Int
}

func NewProofOfWork(b *Block) *ProofOfWork {
	target := big.NewInt(1)

	// Güvenli zorluk (G115 fix + mnd temizliği)
	diff := b.Difficulty
	if diff < 0 {
		diff = 0
	}
	if diff > maxBits {
		diff = maxBits
	}
	shift := uint(hashBits - diff) // burada artık negatif/taşma yok
	target.Lsh(target, shift)

	return &ProofOfWork{block: b, target: target}
}

func (pow *ProofOfWork) prepareData(nonce int) []byte {
	return bytes.Join([][]byte{
		pow.block.PrevHash,
		pow.block.HashTransactions(),
		intToHex(pow.block.Timestamp),
		intToHex(int64(pow.block.Difficulty)),
		intToHex(int64(nonce)),
	}, []byte{})
}

func (pow *ProofOfWork) Run() (int, []byte) {
	var (
		hash    [32]byte
		hashInt big.Int
		nonce   = 0
	)

	for nonce < maxNonce {
		data := pow.prepareData(nonce)
		hash = sha256.Sum256(data)
		hashInt.SetBytes(hash[:])

		if hashInt.Cmp(pow.target) == -1 {
			break
		}
		nonce++
	}
	return nonce, hash[:]
}

func (pow *ProofOfWork) Validate() bool {
	var hashInt big.Int
	data := pow.prepareData(pow.block.Nonce)
	sum := sha256.Sum256(data)
	hashInt.SetBytes(sum[:])

	return hashInt.Cmp(pow.target) == -1
}

// intToHex eski yardımcı (mevcut projede zaten vardı)
func intToHex(num int64) []byte {
	return []byte{
		byte(num >> 56), byte(num >> 48), byte(num >> 40), byte(num >> 32),
		byte(num >> 24), byte(num >> 16), byte(num >> 8), byte(num),
	}
}
