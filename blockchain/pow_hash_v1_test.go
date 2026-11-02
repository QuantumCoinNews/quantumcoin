package blockchain

import (
	"bytes"
	"crypto/sha256"
	"testing"
	"time"
)

func canonicalPoWTestBlockV1(
	difficulty int,
) *Block {
	tx := &Transaction{
		ID:     nil,
		Inputs: []TransactionInput{},
		Outputs: []TransactionOutput{
			{
				Amount:     1,
				PubKeyHash: []byte("pow-test-recipient"),
			},
		},
		Timestamp: time.Unix(1700000000, 0).UTC(),
		Sender:    "COINBASE",
		Amount:    1,
	}

	tx.ID = tx.Hash()

	return &Block{
		Index:        1,
		Timestamp:    1700000001,
		Transactions: []*Transaction{tx},
		PrevHash: bytes.Repeat(
			[]byte{0x42},
			32,
		),
		Hash:       nil,
		Nonce:      0,
		Miner:      "QC_POW_TEST_MINER",
		Difficulty: difficulty,
		Metadata: map[string]string{
			"test": QCCanonicalPoWHashVersionV1,
		},
	}
}

func TestComputePoWHashV1MatchesConsensusPrepareData(
	t *testing.T,
) {
	block := canonicalPoWTestBlockV1(8)
	nonce := 12345

	got, err := ComputePoWHashV1(
		block,
		nonce,
	)
	if err != nil {
		t.Fatalf("canonical hash failed: %v", err)
	}

	pow := NewProofOfWork(block)
	data, err := pow.prepareData(nonce)
	if err != nil {
		t.Fatalf("prepare consensus data: %v", err)
	}
	want := sha256.Sum256(data)

	if got != want {
		t.Fatalf(
			"canonical hash mismatch: got %x want %x",
			got,
			want,
		)
	}
}

func TestPoWHashMeetsTargetV1MatchesBlockValidation(
	t *testing.T,
) {
	block := canonicalPoWTestBlockV1(8)

	found := false

	for nonce := 0; nonce < 1_000_000; nonce++ {
		ok, hash, err := PoWHashMeetsTargetV1(
			block,
			nonce,
		)
		if err != nil {
			t.Fatalf(
				"target check failed at nonce %d: %v",
				nonce,
				err,
			)
		}

		if !ok {
			continue
		}

		block.Nonce = nonce
		block.Hash = append(
			[]byte(nil),
			hash[:]...,
		)

		if !block.ValidatePoW() {
			t.Fatalf(
				"canonical accepted nonce %d but Block.ValidatePoW rejected it",
				nonce,
			)
		}

		found = true
		break
	}

	if !found {
		t.Fatal(
			"no valid difficulty-8 nonce found within test range",
		)
	}
}

func TestComputePoWHashV1RejectsNilBlock(
	t *testing.T,
) {
	if _, err := ComputePoWHashV1(
		nil,
		0,
	); err == nil {
		t.Fatal("nil block was accepted")
	}
}

func TestPoWHashMeetsTargetV1RejectsNilBlock(
	t *testing.T,
) {
	if _, _, err := PoWHashMeetsTargetV1(
		nil,
		0,
	); err == nil {
		t.Fatal("nil block was accepted")
	}
}
