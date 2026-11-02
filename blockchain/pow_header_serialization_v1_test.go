package blockchain

import (
	"bytes"
	"crypto/sha256"
	"testing"
)

func powHeaderSerializationTestBlockV1() *Block {
	return &Block{
		Index:        17,
		Timestamp:    1_725_000_123,
		Transactions: nil,
		PrevHash: []byte{
			0x01, 0x02, 0x03, 0x04,
			0x05, 0x06, 0x07, 0x08,
		},
		Hash:       nil,
		Nonce:      0,
		Miner:      "not-hashed-in-v1",
		Difficulty: 8,
		Metadata: map[string]string{
			"not": "hashed-in-v1",
		},
	}
}

func TestSerializePoWHeaderV1MatchesHistoricalPreimage(
	t *testing.T,
) {
	block := powHeaderSerializationTestBlockV1()
	nonce := 12345

	got, err := SerializePoWHeaderV1(block, nonce)
	if err != nil {
		t.Fatalf("serialize canonical header: %v", err)
	}

	want := bytes.Join(
		[][]byte{
			block.PrevHash,
			block.HashTransactions(),
			intToHex(block.Timestamp),
			intToHex(int64(block.Difficulty)),
			intToHex(int64(nonce)),
		},
		nil,
	)

	if !bytes.Equal(got, want) {
		t.Fatalf(
			"canonical header changed historical bytes:\ngot  %x\nwant %x",
			got,
			want,
		)
	}

	expectedLength := len(block.PrevHash) + sha256.Size + 24
	if len(got) != expectedLength {
		t.Fatalf(
			"canonical header length: got %d want %d",
			len(got),
			expectedLength,
		)
	}
}

func TestSerializePoWHeaderV1IsDeterministic(
	t *testing.T,
) {
	block := powHeaderSerializationTestBlockV1()

	first, err := SerializePoWHeaderV1(block, 987654)
	if err != nil {
		t.Fatalf("first serialization: %v", err)
	}

	second, err := SerializePoWHeaderV1(block, 987654)
	if err != nil {
		t.Fatalf("second serialization: %v", err)
	}

	if !bytes.Equal(first, second) {
		t.Fatalf(
			"canonical serialization is not deterministic:\n%x\n%x",
			first,
			second,
		)
	}
}

func TestSerializePoWHeaderV1RejectsNegativeNonce(
	t *testing.T,
) {
	block := powHeaderSerializationTestBlockV1()

	if _, err := SerializePoWHeaderV1(block, -1); err == nil {
		t.Fatalf("expected negative nonce rejection")
	}
}

func TestValidateBlockPoWV1RequiresStoredHashMatch(
	t *testing.T,
) {
	block := powHeaderSerializationTestBlockV1()
	block.Difficulty = 8

	found := false

	for nonce := 0; nonce < 1_000_000; nonce++ {
		ok, hash, err := PoWHashMeetsTargetV1(block, nonce)
		if err != nil {
			t.Fatalf("canonical PoW check: %v", err)
		}

		if !ok {
			continue
		}

		block.Nonce = nonce
		block.Hash = append([]byte(nil), hash[:]...)
		found = true

		break
	}

	if !found {
		t.Fatalf("no valid nonce found within test range")
	}

	if err := ValidateBlockPoWV1(block); err != nil {
		t.Fatalf("valid stored PoW rejected: %v", err)
	}

	if !block.ValidatePoW() {
		t.Fatalf("Block.ValidatePoW rejected valid canonical PoW")
	}

	block.Hash = append([]byte(nil), block.Hash...)
	block.Hash[0] ^= 0xff

	if err := ValidateBlockPoWV1(block); err == nil {
		t.Fatalf("expected stored hash mismatch rejection")
	}

	if block.ValidatePoW() {
		t.Fatalf(
			"Block.ValidatePoW accepted a mismatched stored hash",
		)
	}
}

func TestValidateBlockPoWV1RejectsMissingStoredHash(
	t *testing.T,
) {
	block := powHeaderSerializationTestBlockV1()

	if err := ValidateBlockPoWV1(block); err == nil {
		t.Fatalf("expected missing stored hash rejection")
	}
}
