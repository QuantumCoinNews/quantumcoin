package miner

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"quantumcoin/blockchain"
)

func workerCanonicalBlockV1(
	difficulty int,
) *blockchain.Block {
	tx := &blockchain.Transaction{
		ID:     nil,
		Inputs: []blockchain.TransactionInput{},
		Outputs: []blockchain.TransactionOutput{
			{
				Amount:     1,
				PubKeyHash: []byte("worker-pow-recipient"),
			},
		},
		Timestamp: time.Unix(
			1700000100,
			0,
		).UTC(),
		Sender: "COINBASE",
		Amount: 1,
	}

	tx.ID = tx.Hash()

	return &blockchain.Block{
		Index:        1,
		Timestamp:    1700000101,
		Transactions: []*blockchain.Transaction{tx},
		PrevHash: bytes.Repeat(
			[]byte{0x24},
			32,
		),
		Hash:       nil,
		Nonce:      0,
		Miner:      "QC_WORKER_TEST_MINER",
		Difficulty: difficulty,
		Metadata: map[string]string{
			"test": "worker-canonical-pow-v1",
		},
	}
}

func TestHashWorkCandidateV1CanonicalMatchesBlockchain(
	t *testing.T,
) {
	block := workerCanonicalBlockV1(8)

	work := &Work{
		Target:   targetFromBits(8),
		Height:   block.Index,
		Miner:    block.Miner,
		Template: block,
	}

	nonce := uint64(12345)

	gotOK, gotHash, err :=
		hashWorkCandidateV1(
			work,
			nonce,
		)
	if err != nil {
		t.Fatalf(
			"worker canonical hash failed: %v",
			err,
		)
	}

	wantOK, wantHash, err :=
		blockchain.PoWHashMeetsTargetV1(
			block,
			int(nonce),
		)
	if err != nil {
		t.Fatalf(
			"blockchain canonical hash failed: %v",
			err,
		)
	}

	if gotOK != wantOK {
		t.Fatalf(
			"target result mismatch: got %v want %v",
			gotOK,
			wantOK,
		)
	}

	if gotHash != wantHash {
		t.Fatalf(
			"hash mismatch: got %x want %x",
			gotHash,
			wantHash,
		)
	}
}

func TestHashWorkCandidateV1LegacyMatchesHashCandidate(
	t *testing.T,
) {
	target := new(big.Int).Lsh(
		big.NewInt(1),
		256,
	)

	work := &Work{
		Left:   []byte("legacy-left"),
		Right:  []byte("legacy-right"),
		Target: target,
		Height: 1,
		Miner:  "QC_LEGACY_TEST",
	}

	nonce := uint64(42)

	gotOK, gotHash, err :=
		hashWorkCandidateV1(
			work,
			nonce,
		)
	if err != nil {
		t.Fatalf(
			"legacy worker hash failed: %v",
			err,
		)
	}

	wantHash := hashCandidate(
		work.Left,
		nonce,
		work.Right,
	)

	if !gotOK {
		t.Fatal("maximum legacy target was rejected")
	}

	if gotHash != wantHash {
		t.Fatalf(
			"legacy hash mismatch: got %x want %x",
			gotHash,
			wantHash,
		)
	}
}

func TestHashWorkCandidateV1RejectsNilWork(
	t *testing.T,
) {
	if _, _, err := hashWorkCandidateV1(
		nil,
		0,
	); err == nil {
		t.Fatal("nil work was accepted")
	}
}

func TestHashWorkCandidateV1RejectsNilTarget(
	t *testing.T,
) {
	work := &Work{
		Template: workerCanonicalBlockV1(8),
	}

	if _, _, err := hashWorkCandidateV1(
		work,
		0,
	); err == nil {
		t.Fatal("nil target was accepted")
	}
}

type workerSubmitErrorBackendV1 struct{}

func (workerSubmitErrorBackendV1) GetWork(
	context.Context,
	string,
) (*Work, error) {
	return nil, nil
}

func (workerSubmitErrorBackendV1) Submit(
	context.Context,
	*Work,
	uint64,
	string,
) (bool, error) {
	return false, errors.New(
		"worker-submit-error-sentinel",
	)
}

func TestWorkerSolveV1PropagatesSubmitError(
	t *testing.T,
) {
	target := new(big.Int).Lsh(
		big.NewInt(1),
		256,
	)

	worker := NewWorker(
		workerSubmitErrorBackendV1{},
		WorkerConfig{
			Threads: 1,
		},
	)

	err := worker.solve(
		context.Background(),
		&Work{
			Left:   []byte("submit-left"),
			Right:  []byte("submit-right"),
			Target: target,
			Height: 1,
			Miner:  "QC_SUBMIT_TEST",
		},
	)

	if err == nil {
		t.Fatal("backend submit error was ignored")
	}

	if !strings.Contains(
		err.Error(),
		"worker-submit-error-sentinel",
	) {
		t.Fatalf(
			"unexpected submit error: %v",
			err,
		)
	}
}
