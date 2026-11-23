package wallet

import (
	"crypto/sha256"
	"quantumcoin/utils"
	"strings"
	"testing"
)

func TestValidateAddressBoundsAndCanonicalVersion(t *testing.T) {
	valid := NewWallet().GetAddress()
	if !ValidateAddress(valid) {
		t.Fatal("generated address rejected")
	}
	raw := append([]byte{0x6f}, make([]byte, 20)...)
	first := sha256.Sum256(raw)
	second := sha256.Sum256(first[:])
	foreign := string(utils.Base58Encode(append(raw, second[:4]...)))
	for _, value := range []string{"", "1", "@@@", foreign, " " + valid, valid + " ", strings.Repeat("1", 36), strings.Repeat("z", 4096)} {
		if ValidateAddress(value) {
			t.Fatalf("invalid address accepted (length=%d)", len(value))
		}
	}
	oversized := strings.Repeat("z", 4096)
	if allocations := testing.AllocsPerRun(100, func() {
		if ValidateAddress(oversized) {
			t.Fatal("oversized accepted")
		}
	}); allocations != 0 {
		t.Fatalf("oversized input reached allocating decoder: allocations=%v", allocations)
	}
}
