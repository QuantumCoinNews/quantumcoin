package blockchain

import (
	"bytes"
	"strconv"
	"testing"
)

func nonceAPITestBlockV1() *Block {
	return &Block{
		Index:        25,
		Timestamp:    1_725_000_999,
		Transactions: nil,
		PrevHash: []byte{
			0x10, 0x20, 0x30, 0x40,
			0x50, 0x60, 0x70, 0x80,
		},
		Hash:       nil,
		Nonce:      0,
		Miner:      "uint64-api-test",
		Difficulty: 8,
		Metadata:   map[string]string{},
	}
}

func TestConsensusNonceIntV1Boundaries(
	t *testing.T,
) {
	if strconv.IntSize != 64 {
		if _, err := ConsensusNonceIntV1(0); err == nil {
			t.Fatalf("expected non-64-bit runtime rejection")
		}

		return
	}

	got, err := ConsensusNonceIntV1(
		MaxConsensusNonceV1,
	)
	if err != nil {
		t.Fatalf("maximum nonce rejected: %v", err)
	}

	if uint64(got) != MaxConsensusNonceV1 {
		t.Fatalf(
			"nonce conversion: got %d want %d",
			got,
			MaxConsensusNonceV1,
		)
	}

	if _, err := ConsensusNonceIntV1(
		MaxConsensusNonceV1 + 1,
	); err == nil {
		t.Fatalf("expected nonce overflow rejection")
	}
}

func TestSerializePoWHeaderUint64V1MatchesCanonicalV1(
	t *testing.T,
) {
	block := nonceAPITestBlockV1()
	nonce := uint64(12345)

	got, err := SerializePoWHeaderUint64V1(
		block,
		nonce,
	)
	if err != nil {
		t.Fatalf("uint64 serialization failed: %v", err)
	}

	want, err := SerializePoWHeaderV1(
		block,
		int(nonce),
	)
	if err != nil {
		t.Fatalf("historical serialization failed: %v", err)
	}

	if !bytes.Equal(got, want) {
		t.Fatalf(
			"uint64 header differs from canonical header:\ngot  %x\nwant %x",
			got,
			want,
		)
	}

	if len(got) < 8 {
		t.Fatalf(
			"serialized header is too short: %d",
			len(got),
		)
	}

	wantNonceBytes := []byte{
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x30, 0x39,
	}

	if !bytes.Equal(
		got[len(got)-8:],
		wantNonceBytes,
	) {
		t.Fatalf(
			"nonce is not fixed-width big-endian: %x",
			got[len(got)-8:],
		)
	}
}

func TestComputePoWHashUint64V1MatchesCanonicalV1(
	t *testing.T,
) {
	block := nonceAPITestBlockV1()
	nonce := uint64(987654)

	got, err := ComputePoWHashUint64V1(
		block,
		nonce,
	)
	if err != nil {
		t.Fatalf("uint64 hash failed: %v", err)
	}

	want, err := ComputePoWHashV1(
		block,
		int(nonce),
	)
	if err != nil {
		t.Fatalf("canonical hash failed: %v", err)
	}

	if got != want {
		t.Fatalf(
			"uint64 hash mismatch: got %x want %x",
			got,
			want,
		)
	}
}

func TestPoWHashMeetsTargetUint64V1MatchesCanonicalV1(
	t *testing.T,
) {
	block := nonceAPITestBlockV1()
	nonce := uint64(54321)

	gotOK, gotHash, err :=
		PoWHashMeetsTargetUint64V1(
			block,
			nonce,
		)
	if err != nil {
		t.Fatalf("uint64 target check failed: %v", err)
	}

	wantOK, wantHash, err :=
		PoWHashMeetsTargetV1(
			block,
			int(nonce),
		)
	if err != nil {
		t.Fatalf("canonical target check failed: %v", err)
	}

	if gotOK != wantOK || gotHash != wantHash {
		t.Fatalf(
			"uint64 target mismatch: got=(%v,%x) want=(%v,%x)",
			gotOK,
			gotHash,
			wantOK,
			wantHash,
		)
	}
}

func TestUint64PoWAPIRejectsNonceOverflow(
	t *testing.T,
) {
	block := nonceAPITestBlockV1()
	tooLarge := MaxConsensusNonceV1 + 1

	if _, err := SerializePoWHeaderUint64V1(
		block,
		tooLarge,
	); err == nil {
		t.Fatalf("expected serializer overflow rejection")
	}

	if _, err := ComputePoWHashUint64V1(
		block,
		tooLarge,
	); err == nil {
		t.Fatalf("expected hash overflow rejection")
	}

	if _, _, err := PoWHashMeetsTargetUint64V1(
		block,
		tooLarge,
	); err == nil {
		t.Fatalf("expected target overflow rejection")
	}
}
