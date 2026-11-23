package wallet

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	secpECDSA "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"strings"
	"testing"
)

func TestRecoveryLinkPublicProofAndScope(t *testing.T) {
	m, err := RestoreRecoveryMaterial(strings.Repeat("abandon ", 11)+"about", "")
	if err != nil {
		t.Fatal(err)
	}
	w, _ := m.Wallet()
	c := RecoveryLinkChallenge{Version: "QC_GAME_WALLET_LINK_V1", NetworkID: "testnet", GenesisHex: strings.Repeat("1", 64), PlayerID: "tg_42", Address: m.Address, WalletOwner: hex.EncodeToString(HashPubKey(w.PublicKey)), Nonce: strings.Repeat("2", 64), IssuedAt: 1000, ExpiresAt: 1300}
	proof, err := SignRecoveryLink(*m, c, "tg_42", 1100)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(c)
	var fields map[string]any
	_ = json.Unmarshal(b, &fields)
	canonical, _ := json.Marshal(fields)
	digest := sha256.Sum256(append([]byte("QC_GAME_WALLET_LINK_V1\x00"), canonical...))
	raw, _ := hex.DecodeString(proof.Signature)
	signature, err := secpECDSA.ParseDERSignature(raw)
	if err != nil {
		t.Fatal(err)
	}
	public, err := secp256k1.ParsePubKey(w.PublicKey)
	if err != nil || !signature.Verify(digest[:], public) {
		t.Fatal("invalid ownership proof")
	}
	exposed, _ := json.Marshal(proof)
	if strings.Contains(string(exposed), "abandon") || strings.Contains(string(exposed), "private") {
		t.Fatal("secret in proof")
	}
	for _, change := range []func(*RecoveryLinkChallenge){func(c *RecoveryLinkChallenge) { c.PlayerID = "tg_43" }, func(c *RecoveryLinkChallenge) { c.ExpiresAt = 1100 }, func(c *RecoveryLinkChallenge) { c.GenesisHex = strings.Repeat("0", 64) }, func(c *RecoveryLinkChallenge) { c.WalletOwner = strings.Repeat("0", 40) }, func(c *RecoveryLinkChallenge) { c.Version = "TX" }} {
		altered := c
		change(&altered)
		if _, err := SignRecoveryLink(*m, altered, "tg_42", 1100); err == nil {
			t.Fatal("invalid scope accepted")
		}
	}
}
