package blockchain

import (
	"bytes"
	"testing"

	qconsensus "quantumcoin/consensus"
)

// QMA-0062:
// The canonical burn sink is a consensus marker, not a normal wallet address.
func TestCanonicalBurnSinkIsNotNormalWalletOutputV1(t *testing.T) {
	if qconsensus.CanonicalBurnSinkAddressV1 == "" {
		t.Fatal("canonical burn sink must not be empty")
	}

	if !IsCanonicalBurnAddressV1(
		qconsensus.CanonicalBurnSinkAddressV1,
	) {
		t.Fatal("canonical burn sink must be recognized by consensus")
	}
}

// QMA-0062:
// A canonical burn output must never unlock for a normal wallet key hash.
func TestCanonicalBurnOutputCannotBeUnlockedV1(t *testing.T) {
	out := NewCanonicalBurnOutputV1(1)

	if out.Amount != 1 {
		t.Fatalf(
			"burn output amount mismatch: got=%d want=1",
			out.Amount,
		)
	}

	if !IsCanonicalBurnOutputV1(out) {
		t.Fatal("output must be classified as canonical burn")
	}

	candidates := [][]byte{
		nil,
		{},
		{0x00},
		bytes.Repeat([]byte{0x00}, 20),
		bytes.Repeat([]byte{0x01}, 20),
		bytes.Repeat([]byte{0xff}, 20),
	}

	for _, candidate := range candidates {
		if out.IsLockedWithKey(candidate) {
			t.Fatalf(
				"canonical burn output became spendable with pubKeyHash=%x",
				candidate,
			)
		}
	}
}
