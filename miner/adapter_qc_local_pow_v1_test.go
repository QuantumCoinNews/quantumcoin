package miner

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"quantumcoin/blockchain"
)

func qCLocalAdapterTemplateForTestV1(
	difficulty int,
) *blockchain.Block {
	coinbase := &blockchain.Transaction{
		ID:     nil,
		Inputs: []blockchain.TransactionInput{},
		Outputs: []blockchain.TransactionOutput{
			{
				Amount:     1,
				PubKeyHash: []byte("adapter-test-recipient"),
			},
		},
		Timestamp: time.Unix(
			1700000200,
			0,
		).UTC(),
		Sender: "COINBASE",
		Amount: 1,
	}

	coinbase.ID = coinbase.Hash()

	return &blockchain.Block{
		Index:     1,
		Timestamp: 1700000201,
		Transactions: []*blockchain.Transaction{
			coinbase,
		},
		PrevHash: bytes.Repeat(
			[]byte{0x37},
			32,
		),
		Hash:       nil,
		Nonce:      0,
		Miner:      "QC_ADAPTER_TEST_MINER",
		Difficulty: difficulty,
		Metadata: map[string]string{
			"test": "qc-local-adapter-template-v1",
		},
	}
}

func qCLocalAdapterValidWorkV1(
	t *testing.T,
) (
	*blockchain.Block,
	*blockchain.Block,
	*Work,
	uint64,
	string,
) {
	t.Helper()

	template := qCLocalAdapterTemplateForTestV1(8)

	token, err := qCLocalTemplateTokenV1(
		template,
	)
	if err != nil {
		t.Fatalf(
			"template token failed: %v",
			err,
		)
	}

	work := &Work{
		Left:   []byte(token),
		Target: targetFromBits(8),
		Height: template.Index,
		Miner:  template.Miner,
		Template: cloneQCLocalTemplateV1(
			template,
		),
	}

	last := &blockchain.Block{
		Index: template.Index - 1,
		Hash: append(
			[]byte(nil),
			template.PrevHash...,
		),
	}

	for nonce := 0; nonce < 1_000_000; nonce++ {
		ok, hash, err :=
			blockchain.PoWHashMeetsTargetV1(
				work.Template,
				nonce,
			)
		if err != nil {
			t.Fatalf(
				"canonical pow failed: %v",
				err,
			)
		}

		if ok {
			return template,
				last,
				work,
				uint64(nonce),
				hex.EncodeToString(hash[:])
		}
	}

	t.Fatal(
		"valid difficulty-8 nonce not found",
	)

	return nil, nil, nil, 0, ""
}

func TestQCLocalPrepareSubmissionV1AcceptsFixedTemplate(
	t *testing.T,
) {
	template, last, work, nonce, hashHex :=
		qCLocalAdapterValidWorkV1(t)

	block, err := prepareQCLocalSubmissionV1(
		last,
		template,
		work,
		nonce,
		hashHex,
		8,
	)
	if err != nil {
		t.Fatalf(
			"valid template was rejected: %v",
			err,
		)
	}

	if block.Nonce != int(nonce) {
		t.Fatalf(
			"nonce mismatch: got %d want %d",
			block.Nonce,
			nonce,
		)
	}

	if !block.ValidatePoW() {
		t.Fatal(
			"prepared block failed ValidatePoW",
		)
	}

	if template.Nonce != 0 ||
		len(template.Hash) != 0 {
		t.Fatal(
			"issued template was mutated",
		)
	}
}

func TestQCLocalPrepareSubmissionV1RejectsTemplateMutation(
	t *testing.T,
) {
	template, last, work, nonce, hashHex :=
		qCLocalAdapterValidWorkV1(t)

	work.Template.Timestamp++

	_, err := prepareQCLocalSubmissionV1(
		last,
		template,
		work,
		nonce,
		hashHex,
		8,
	)
	if err == nil ||
		!strings.Contains(
			err.Error(),
			"template identity mismatch",
		) {
		t.Fatalf(
			"template mutation error mismatch: %v",
			err,
		)
	}
}

func TestQCLocalPrepareSubmissionV1RejectsStaleWork(
	t *testing.T,
) {
	template, last, work, nonce, hashHex :=
		qCLocalAdapterValidWorkV1(t)

	last.Hash = []byte("different-chain-tip")

	_, err := prepareQCLocalSubmissionV1(
		last,
		template,
		work,
		nonce,
		hashHex,
		8,
	)
	if err == nil ||
		!strings.Contains(
			err.Error(),
			"stale work template",
		) {
		t.Fatalf(
			"stale work error mismatch: %v",
			err,
		)
	}
}

func TestQCLocalPrepareSubmissionV1RejectsHashMismatch(
	t *testing.T,
) {
	template, last, work, nonce, hashHex :=
		qCLocalAdapterValidWorkV1(t)

	hashBytes, err := hex.DecodeString(hashHex)
	if err != nil {
		t.Fatalf(
			"decode valid test hash: %v",
			err,
		)
	}

	hashBytes[0] ^= 0xff

	_, err = prepareQCLocalSubmissionV1(
		last,
		template,
		work,
		nonce,
		hex.EncodeToString(hashBytes),
		8,
	)
	if err == nil ||
		!strings.Contains(
			err.Error(),
			"provided hash mismatch",
		) {
		t.Fatalf(
			"hash mismatch error mismatch: %v",
			err,
		)
	}
}

func TestQCLocalPrepareSubmissionV1RejectsDifficultyDrift(
	t *testing.T,
) {
	template, last, work, nonce, hashHex :=
		qCLocalAdapterValidWorkV1(t)

	_, err := prepareQCLocalSubmissionV1(
		last,
		template,
		work,
		nonce,
		hashHex,
		9,
	)
	if err == nil ||
		!strings.Contains(
			err.Error(),
			"difficulty changed",
		) {
		t.Fatalf(
			"difficulty drift error mismatch: %v",
			err,
		)
	}
}

func TestQCLocalPrepareSubmissionV1RejectsTargetMutation(
	t *testing.T,
) {
	template, last, work, nonce, hashHex :=
		qCLocalAdapterValidWorkV1(t)

	work.Target = targetFromBits(9)

	_, err := prepareQCLocalSubmissionV1(
		last,
		template,
		work,
		nonce,
		hashHex,
		8,
	)
	if err == nil ||
		!strings.Contains(
			err.Error(),
			"work target mismatch",
		) {
		t.Fatalf(
			"target mutation error mismatch: %v",
			err,
		)
	}
}
