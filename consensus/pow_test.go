package consensus

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"testing"
)

func testPoWHeaderV1() PoWHeaderV1 {
	transactionDigest := sha256.Sum256(
		[]byte("quantumcoin-pow-test-transactions"),
	)

	return PoWHeaderV1{
		PreviousHash:      []byte{0x01, 0x02, 0x03, 0x04},
		TransactionDigest: transactionDigest[:],
		Timestamp:         1_700_000_000,
		Difficulty:        8,
	}
}

func TestNormalizePoWDifficultyV1(t *testing.T) {
	tests := []struct {
		name       string
		difficulty int
		want       int
	}{
		{
			name:       "negative difficulty becomes zero",
			difficulty: -1,
			want:       0,
		},
		{
			name:       "zero difficulty remains zero",
			difficulty: 0,
			want:       0,
		},
		{
			name:       "valid difficulty remains unchanged",
			difficulty: 16,
			want:       16,
		},
		{
			name:       "difficulty above maximum becomes 255",
			difficulty: 300,
			want:       MaxDifficultyBits,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := NormalizePoWDifficultyV1(
				test.difficulty,
			)

			if got != test.want {
				t.Fatalf(
					"NormalizePoWDifficultyV1(%d) = %d, want %d",
					test.difficulty,
					got,
					test.want,
				)
			}
		})
	}
}

func TestTargetFromDifficultyV1(t *testing.T) {
	tests := []struct {
		name       string
		difficulty int
		wantBitLen int
	}{
		{
			name:       "difficulty zero",
			difficulty: 0,
			wantBitLen: 257,
		},
		{
			name:       "difficulty eight",
			difficulty: 8,
			wantBitLen: 249,
		},
		{
			name:       "difficulty 255",
			difficulty: 255,
			wantBitLen: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			target := TargetFromDifficultyV1(
				test.difficulty,
			)

			if target == nil {
				t.Fatal("target must not be nil")
			}

			if target.BitLen() != test.wantBitLen {
				t.Fatalf(
					"target bit length = %d, want %d",
					target.BitLen(),
					test.wantBitLen,
				)
			}
		})
	}
}

func TestSerializePoWHeaderV1(t *testing.T) {
	header := testPoWHeaderV1()
	const nonce uint64 = 12_345

	got, err := SerializePoWHeaderV1(
		header,
		nonce,
	)
	if err != nil {
		t.Fatalf("SerializePoWHeaderV1 error: %v", err)
	}

	wantLength :=
		len(header.PreviousHash) +
			sha256.Size +
			8 +
			8 +
			8

	if len(got) != wantLength {
		t.Fatalf(
			"serialized length = %d, want %d",
			len(got),
			wantLength,
		)
	}

	offset := 0

	if !bytes.Equal(
		got[offset:offset+len(header.PreviousHash)],
		header.PreviousHash,
	) {
		t.Fatal("previous hash bytes changed")
	}

	offset += len(header.PreviousHash)

	if !bytes.Equal(
		got[offset:offset+sha256.Size],
		header.TransactionDigest,
	) {
		t.Fatal("transaction digest bytes changed")
	}

	offset += sha256.Size

	gotTimestamp := int64(
		binary.BigEndian.Uint64(
			got[offset : offset+8],
		),
	)

	if gotTimestamp != header.Timestamp {
		t.Fatalf(
			"timestamp = %d, want %d",
			gotTimestamp,
			header.Timestamp,
		)
	}

	offset += 8

	gotDifficulty := int64(
		binary.BigEndian.Uint64(
			got[offset : offset+8],
		),
	)

	if gotDifficulty != int64(header.Difficulty) {
		t.Fatalf(
			"difficulty = %d, want %d",
			gotDifficulty,
			header.Difficulty,
		)
	}

	offset += 8

	gotNonce := binary.BigEndian.Uint64(
		got[offset : offset+8],
	)

	if gotNonce != nonce {
		t.Fatalf(
			"nonce = %d, want %d",
			gotNonce,
			nonce,
		)
	}
}

func TestSerializePoWHeaderV1RejectsInvalidDigest(t *testing.T) {
	header := testPoWHeaderV1()
	header.TransactionDigest = []byte("invalid")

	_, err := SerializePoWHeaderV1(
		header,
		0,
	)
	if err == nil {
		t.Fatal("invalid transaction digest must be rejected")
	}
}

func TestSerializePoWHeaderV1RejectsNonceOverflow(t *testing.T) {
	header := testPoWHeaderV1()

	_, err := SerializePoWHeaderV1(
		header,
		MaxConsensusNonceV1+1,
	)
	if err == nil {
		t.Fatal("nonce overflow must be rejected")
	}
}

func TestComputePoWHashV1MatchesSerializedHeader(t *testing.T) {
	header := testPoWHeaderV1()
	const nonce uint64 = 54_321

	serialized, err := SerializePoWHeaderV1(
		header,
		nonce,
	)
	if err != nil {
		t.Fatalf("serialize header: %v", err)
	}

	want := sha256.Sum256(serialized)

	got, err := ComputePoWHashV1(
		header,
		nonce,
	)
	if err != nil {
		t.Fatalf("compute hash: %v", err)
	}

	if got != want {
		t.Fatalf(
			"computed hash = %x, want %x",
			got,
			want,
		)
	}
}

func TestPoWHashMeetsTargetV1AtDifficultyZero(t *testing.T) {
	header := testPoWHeaderV1()
	header.Difficulty = 0

	meetsTarget, hash, err := PoWHashMeetsTargetV1(
		header,
		7,
	)
	if err != nil {
		t.Fatalf("target check: %v", err)
	}

	if !meetsTarget {
		t.Fatal("difficulty-zero hash must meet target")
	}

	if hash == ([32]byte{}) {
		t.Fatal("computed hash must not be zero")
	}
}

func TestValidateStoredPoWHashV1(t *testing.T) {
	header := testPoWHeaderV1()
	header.Difficulty = 0

	const nonce uint64 = 99

	hash, err := ComputePoWHashV1(
		header,
		nonce,
	)
	if err != nil {
		t.Fatalf("compute stored hash: %v", err)
	}

	if err := ValidateStoredPoWHashV1(
		header,
		nonce,
		hash[:],
	); err != nil {
		t.Fatalf("valid stored PoW hash rejected: %v", err)
	}
}

func TestValidateStoredPoWHashV1RejectsMutation(t *testing.T) {
	header := testPoWHeaderV1()
	header.Difficulty = 0

	const nonce uint64 = 101

	hash, err := ComputePoWHashV1(
		header,
		nonce,
	)
	if err != nil {
		t.Fatalf("compute stored hash: %v", err)
	}

	mutated := append([]byte(nil), hash[:]...)
	mutated[0] ^= 0xff

	if err := ValidateStoredPoWHashV1(
		header,
		nonce,
		mutated,
	); err == nil {
		t.Fatal("mutated stored hash must be rejected")
	}
}

func TestValidateStoredPoWHashV1RejectsInvalidLength(t *testing.T) {
	header := testPoWHeaderV1()
	header.Difficulty = 0

	err := ValidateStoredPoWHashV1(
		header,
		0,
		[]byte{0x01, 0x02},
	)
	if err == nil {
		t.Fatal("invalid stored hash length must be rejected")
	}
}
