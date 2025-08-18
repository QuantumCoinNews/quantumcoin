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

func NewProofOfWork(block *Block) *ProofOfWork {
	diff := block.Difficulty
	if diff <= 0 {
		diff = defaultDifficultyBits
	}
	if diff > 255 {
		diff = 255
	}
	target := big.NewInt(1)
	target.Lsh(target, uint(256-diff))
	return &ProofOfWork{Block: block, Target: target, Difficulty: diff}
}

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

func (pow *ProofOfWork) prepareData(nonce int) []byte {
	data := bytes.Join([][]byte{
		pow.Block.PrevHash,
		pow.Block.HashTransactions(),
		[]byte(strconv.Itoa(pow.Block.Index)),
		[]byte(strconv.FormatInt(pow.Block.Timestamp, 10)),
		[]byte(strconv.Itoa(nonce)),
		[]byte(strconv.Itoa(pow.Difficulty)),
		[]byte(pow.Block.Miner),
	}, []byte{})
	return data
}

func (pow *ProofOfWork) Validate() bool {
	var hashInt big.Int
	data := pow.prepareData(pow.Block.Nonce)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])
	return hashInt.Cmp(pow.Target) == -1
}
