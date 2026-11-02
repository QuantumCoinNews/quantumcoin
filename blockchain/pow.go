package blockchain

import (
	"fmt"
	"math/big"

	qconsensus "quantumcoin/consensus"
)

const (
	hashBits = qconsensus.PoWHashBitsV1
	maxBits  = qconsensus.MaxDifficultyBits
)

type ProofOfWork struct {
	block  *Block
	target *big.Int
}

func NewProofOfWork(
	b *Block,
) *ProofOfWork {
	if b == nil {
		return &ProofOfWork{}
	}

	return &ProofOfWork{
		block: b,
		target: qconsensus.TargetFromDifficultyV1(
			b.Difficulty,
		),
	}
}

func (pow *ProofOfWork) prepareData(
	nonce int,
) ([]byte, error) {
	if pow == nil || pow.block == nil {
		return nil, fmt.Errorf(
			"%s: proof of work block is unavailable",
			QCCanonicalPoWHashVersionV1,
		)
	}

	return SerializePoWHeaderV1(
		pow.block,
		nonce,
	)
}

func (pow *ProofOfWork) runNonceRangeV1(
	startNonce uint64,
	maxNonce uint64,
) (int, []byte, error) {
	if pow == nil || pow.block == nil {
		return 0, nil, fmt.Errorf(
			"%s: proof of work block is unavailable",
			QCCanonicalPoWHashVersionV1,
		)
	}

	if pow.target == nil {
		return 0, nil, fmt.Errorf(
			"%s: proof of work target is unavailable",
			QCCanonicalPoWHashVersionV1,
		)
	}

	if startNonce > maxNonce {
		return 0, nil, fmt.Errorf(
			"%s: invalid nonce range %d..%d",
			QCCanonicalPoWHashVersionV1,
			startNonce,
			maxNonce,
		)
	}

	if maxNonce > MaxConsensusNonceV1 {
		return 0, nil, fmt.Errorf(
			"%s: maximum nonce %d exceeds consensus maximum %d",
			QCCanonicalPoWHashVersionV1,
			maxNonce,
			MaxConsensusNonceV1,
		)
	}

	for nonce := startNonce; nonce <= maxNonce; nonce++ {
		hash, err := ComputePoWHashUint64V1(
			pow.block,
			nonce,
		)
		if err != nil {
			return 0, nil, fmt.Errorf(
				"%s: compute nonce %d: %w",
				QCCanonicalPoWHashVersionV1,
				nonce,
				err,
			)
		}

		var hashInt big.Int
		hashInt.SetBytes(
			hash[:],
		)

		if hashInt.Cmp(pow.target) == -1 {
			nonceInt, err := ConsensusNonceIntV1(
				nonce,
			)
			if err != nil {
				return 0, nil, fmt.Errorf(
					"%s: convert nonce %d: %w",
					QCCanonicalPoWHashVersionV1,
					nonce,
					err,
				)
			}

			return nonceInt, append(
				[]byte(nil),
				hash[:]...,
			), nil
		}

		if nonce == maxNonce {
			break
		}
	}

	return 0, nil, fmt.Errorf(
		"%w: searched nonce range %d..%d",
		ErrNonceExhausted,
		startNonce,
		maxNonce,
	)
}

func (pow *ProofOfWork) runWithMaxNonceV1(
	maxNonce uint64,
) (int, []byte, error) {
	return pow.runNonceRangeV1(
		0,
		maxNonce,
	)
}

func (pow *ProofOfWork) Run() (
	int,
	[]byte,
	error,
) {
	return pow.runWithMaxNonceV1(
		MaxConsensusNonceV1,
	)
}

func (pow *ProofOfWork) ValidateE() error {
	if pow == nil || pow.block == nil {
		return fmt.Errorf(
			"%s: proof of work block is unavailable",
			QCCanonicalPoWHashVersionV1,
		)
	}

	if pow.target == nil {
		return fmt.Errorf(
			"%s: proof of work target is unavailable",
			QCCanonicalPoWHashVersionV1,
		)
	}

	hash, err := ComputePoWHashV1(
		pow.block,
		pow.block.Nonce,
	)
	if err != nil {
		return fmt.Errorf(
			"%s: compute stored nonce: %w",
			QCCanonicalPoWHashVersionV1,
			err,
		)
	}

	var hashInt big.Int
	hashInt.SetBytes(
		hash[:],
	)

	if hashInt.Cmp(pow.target) != -1 {
		return ErrInvalidPoW
	}

	return nil
}

func (pow *ProofOfWork) Validate() bool {
	return pow.ValidateE() == nil
}

// intToHex eski yardımcı; geriye dönük uyumluluk için korunur.
func intToHex(num int64) []byte {
	return []byte{
		byte(num >> 56),
		byte(num >> 48),
		byte(num >> 40),
		byte(num >> 32),
		byte(num >> 24),
		byte(num >> 16),
		byte(num >> 8),
		byte(num),
	}
}
