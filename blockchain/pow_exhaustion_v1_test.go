package blockchain

import (
	"bytes"
	"errors"
	"math/big"
	"testing"
)

func TestProofOfWorkRunWithMaxNonceV1ReturnsMatchingNonceAndHash(
	t *testing.T,
) {
	block := canonicalPoWTestBlockV1(8)
	pow := NewProofOfWork(block)

	nonce, hash, err := pow.runWithMaxNonceV1(1_000_000)
	if err != nil {
		t.Fatalf("mine bounded PoW: %v", err)
	}

	computed, err := ComputePoWHashV1(block, nonce)
	if err != nil {
		t.Fatalf("recompute mined hash: %v", err)
	}

	if !bytes.Equal(hash, computed[:]) {
		t.Fatalf(
			"returned nonce/hash pair does not match: nonce=%d got=%x want=%x",
			nonce,
			hash,
			computed,
		)
	}
}

func TestProofOfWorkRunWithMaxNonceV1ReturnsExplicitExhaustion(
	t *testing.T,
) {
	block := canonicalPoWTestBlockV1(8)
	pow := NewProofOfWork(block)
	pow.target = big.NewInt(0)

	nonce, hash, err := pow.runWithMaxNonceV1(3)
	if !errors.Is(err, ErrNonceExhausted) {
		t.Fatalf(
			"expected ErrNonceExhausted, got nonce=%d hash=%x err=%v",
			nonce,
			hash,
			err,
		)
	}

	if nonce != 0 {
		t.Fatalf("exhaustion returned a fake nonce: %d", nonce)
	}

	if hash != nil {
		t.Fatalf("exhaustion returned a fake hash: %x", hash)
	}
}

func TestProofOfWorkRunNonceRangeV1AcceptsLastAllowedNonce(
	t *testing.T,
) {
	block := canonicalPoWTestBlockV1(8)
	const lastNonce uint64 = 7

	wantHash, err := ComputePoWHashUint64V1(
		block,
		lastNonce,
	)
	if err != nil {
		t.Fatalf("compute last allowed nonce: %v", err)
	}

	pow := NewProofOfWork(block)
	pow.target = new(big.Int).Add(
		new(big.Int).SetBytes(wantHash[:]),
		big.NewInt(1),
	)

	gotNonce, gotHash, err := pow.runNonceRangeV1(
		lastNonce,
		lastNonce,
	)
	if err != nil {
		t.Fatalf("last allowed nonce rejected: %v", err)
	}

	if uint64(gotNonce) != lastNonce {
		t.Fatalf(
			"last allowed nonce mismatch: got %d want %d",
			gotNonce,
			lastNonce,
		)
	}

	if !bytes.Equal(gotHash, wantHash[:]) {
		t.Fatalf(
			"last allowed nonce hash mismatch: got=%x want=%x",
			gotHash,
			wantHash,
		)
	}
}

func TestProofOfWorkRunWithMaxNonceV1PropagatesPreparationErrors(
	t *testing.T,
) {
	pow := NewProofOfWork(nil)

	_, _, err := pow.runWithMaxNonceV1(0)
	if err == nil {
		t.Fatalf("expected nil-block preparation error")
	}

	if errors.Is(err, ErrNonceExhausted) {
		t.Fatalf("preparation error was misreported as exhaustion: %v", err)
	}
}

func TestProofOfWorkRunWithMaxNonceV1RejectsNonceLimitOverflow(
	t *testing.T,
) {
	block := canonicalPoWTestBlockV1(8)
	pow := NewProofOfWork(block)

	_, _, err := pow.runWithMaxNonceV1(
		MaxConsensusNonceV1 + 1,
	)
	if err == nil {
		t.Fatalf("expected maximum nonce overflow rejection")
	}

	if errors.Is(err, ErrNonceExhausted) {
		t.Fatalf("nonce limit overflow was misreported as exhaustion: %v", err)
	}
}
