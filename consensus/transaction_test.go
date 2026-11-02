package consensus_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/consensus"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

func transactionFixtureV1() *blockchain.Transaction {
	return &blockchain.Transaction{
		ID: nil,
		Inputs: []blockchain.TransactionInput{
			{
				TxID: bytes.Repeat(
					[]byte{0x11},
					sha256.Size,
				),
				OutIndex: 2,
			},
		},
		Outputs: []blockchain.TransactionOutput{
			{
				Amount: 7,
				PubKeyHash: []byte{
					0x21, 0x22, 0x23, 0x24,
				},
			},
			{
				Amount: 3,
				PubKeyHash: []byte{
					0x31, 0x32, 0x33, 0x34,
				},
			},
		},
		Timestamp: time.Unix(
			1_700_000_000,
			0,
		).UTC(),
		Sender: "quantumcoin-test-sender",
		Amount: 7,
	}
}

func canonicalTransactionPayloadV1(
	tx *blockchain.Transaction,
) ([]byte, error) {
	copyTx := tx.TrimmedCopy()
	copyTx.ID = nil

	var buffer bytes.Buffer

	if err := gob.NewEncoder(&buffer).Encode(
		&copyTx,
	); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func TestCanonicalTransactionIDCompatibilityV1(
	t *testing.T,
) {
	tx := transactionFixtureV1()

	blockchainID, err :=
		blockchain.CanonicalTransactionIDV1(tx)
	if err != nil {
		t.Fatalf(
			"blockchain canonical transaction ID: %v",
			err,
		)
	}

	payload, err := canonicalTransactionPayloadV1(tx)
	if err != nil {
		t.Fatalf(
			"canonical transaction payload: %v",
			err,
		)
	}

	consensusID, err :=
		consensus.CanonicalTransactionIDFromPayloadV1(
			payload,
		)
	if err != nil {
		t.Fatalf(
			"consensus canonical transaction ID: %v",
			err,
		)
	}

	if !bytes.Equal(
		blockchainID,
		consensusID[:],
	) {
		t.Fatalf(
			"transaction ID mismatch:\nblockchain=%x\nconsensus=%x",
			blockchainID,
			consensusID,
		)
	}

	if err := consensus.ValidateTransactionIDFromPayloadV1(
		blockchainID,
		payload,
	); err != nil {
		t.Fatalf(
			"compatible transaction ID rejected: %v",
			err,
		)
	}
}

func TestSigningDigestCompatibilityV1(
	t *testing.T,
) {
	tx := transactionFixtureV1()

	payload, err := canonicalTransactionPayloadV1(tx)
	if err != nil {
		t.Fatalf(
			"canonical transaction payload: %v",
			err,
		)
	}

	consensusDigest, err :=
		consensus.SigningDigestV1(
			payload,
			tx.Inputs[0].TxID,
			tx.Inputs[0].OutIndex,
		)
	if err != nil {
		t.Fatalf(
			"consensus signing digest: %v",
			err,
		)
	}

	blockchainDigests :=
		blockchain.SigningHashes(tx)

	if len(blockchainDigests) != 1 {
		t.Fatalf(
			"blockchain signing digest count = %d, want 1",
			len(blockchainDigests),
		)
	}

	blockchainDigest, err :=
		hex.DecodeString(blockchainDigests[0])
	if err != nil {
		t.Fatalf(
			"decode blockchain signing digest: %v",
			err,
		)
	}

	if !bytes.Equal(
		blockchainDigest,
		consensusDigest[:],
	) {
		t.Fatalf(
			"signing digest mismatch:\nblockchain=%x\nconsensus=%x",
			blockchainDigest,
			consensusDigest,
		)
	}
}

func TestCanonicalSignatureEncodeDecodeV1(
	t *testing.T,
) {
	r := big.NewInt(123_456)
	s := big.NewInt(789_012)

	encoded, err :=
		consensus.EncodeCanonicalTransactionSignatureV1(
			r,
			s,
		)
	if err != nil {
		t.Fatalf(
			"encode canonical signature: %v",
			err,
		)
	}

	decodedR, decodedS, err :=
		consensus.DecodeCanonicalTransactionSignatureV1(
			encoded,
		)
	if err != nil {
		t.Fatalf(
			"decode canonical signature: %v",
			err,
		)
	}

	if decodedR.Cmp(r) != 0 {
		t.Fatalf(
			"decoded r = %s, want %s",
			decodedR,
			r,
		)
	}

	if decodedS.Cmp(s) != 0 {
		t.Fatalf(
			"decoded s = %s, want %s",
			decodedS,
			s,
		)
	}
}

func TestNormalizeTransactionLowSV1(
	t *testing.T,
) {
	order := secp256k1.S256().Params().N

	highS := new(big.Int).Sub(
		order,
		big.NewInt(1),
	)

	normalized :=
		consensus.NormalizeTransactionLowSV1(
			highS,
		)

	if normalized == nil {
		t.Fatal("normalized S must not be nil")
	}

	if normalized.Cmp(big.NewInt(1)) != 0 {
		t.Fatalf(
			"normalized S = %s, want 1",
			normalized,
		)
	}
}

func TestTransactionSignatureRoundTripV1(
	t *testing.T,
) {
	privateKey, err := ecdsa.GenerateKey(
		secp256k1.S256(),
		rand.Reader,
	)
	if err != nil {
		t.Fatalf(
			"generate secp256k1 private key: %v",
			err,
		)
	}

	digest := sha256.Sum256(
		[]byte(
			"quantumcoin-transaction-signature-test",
		),
	)

	signature, err :=
		consensus.SignTransactionDigestV1(
			privateKey,
			digest,
		)
	if err != nil {
		t.Fatalf(
			"sign transaction digest: %v",
			err,
		)
	}

	publicKey, err :=
		consensus.UncompressedPublicKeyV1(
			privateKey,
		)
	if err != nil {
		t.Fatalf(
			"serialize public key: %v",
			err,
		)
	}

	if err := consensus.VerifyTransactionDigestSignatureV1(
		publicKey,
		digest,
		signature,
	); err != nil {
		t.Fatalf(
			"valid transaction signature rejected: %v",
			err,
		)
	}

	mutatedDigest := digest
	mutatedDigest[0] ^= 0xff

	if err := consensus.VerifyTransactionDigestSignatureV1(
		publicKey,
		mutatedDigest,
		signature,
	); err == nil {
		t.Fatal(
			"signature must fail after digest mutation",
		)
	}
}

func TestTransactionIDMutationRejectedV1(
	t *testing.T,
) {
	tx := transactionFixtureV1()

	payload, err := canonicalTransactionPayloadV1(tx)
	if err != nil {
		t.Fatalf(
			"canonical transaction payload: %v",
			err,
		)
	}

	transactionID, err :=
		consensus.CanonicalTransactionIDFromPayloadV1(
			payload,
		)
	if err != nil {
		t.Fatalf(
			"canonical transaction ID: %v",
			err,
		)
	}

	mutatedID := append(
		[]byte(nil),
		transactionID[:]...,
	)

	mutatedID[0] ^= 0xff

	if err := consensus.ValidateTransactionIDFromPayloadV1(
		mutatedID,
		payload,
	); err == nil {
		t.Fatal(
			"mutated transaction ID must be rejected",
		)
	}
}
func TestSigningDigestAllowsLegacyEmptyInputTransactionIDV1(
	t *testing.T,
) {
	trimmedPayload := []byte(
		"legacy-empty-input-transaction-id",
	)

	expectedPayload := append(
		[]byte(nil),
		trimmedPayload...,
	)

	// Legacy signMessageBytes, boş TxID için hiçbir byte eklemeden
	// yalnız outIndex değerini uint32 big-endian olarak ekler.
	expectedPayload = append(
		expectedPayload,
		0x00,
		0x00,
		0x00,
		0x03,
	)

	expectedDigest := sha256.Sum256(expectedPayload)

	actualDigest, err :=
		consensus.SigningDigestV1(
			trimmedPayload,
			nil,
			3,
		)
	if err != nil {
		t.Fatalf(
			"legacy empty input transaction ID rejected: %v",
			err,
		)
	}

	if actualDigest != expectedDigest {
		t.Fatalf(
			"legacy empty input transaction ID digest mismatch:\nactual=%x\nexpected=%x",
			actualDigest,
			expectedDigest,
		)
	}
}
