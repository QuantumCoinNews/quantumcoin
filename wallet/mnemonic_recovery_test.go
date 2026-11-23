package wallet

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Public BIP39 zero-entropy vector, never a production wallet.
func publicRecoveryWords() string { return strings.Repeat("abandon ", 11) + "about" }

func TestMnemonicRecoveryPreservesOldQCWalletAddress(t *testing.T) {
	const expected = "1FLm5EcKtdBGNYg42Q21tfoXFwA5KYZy8W"
	material, err := RestoreRecoveryMaterial(publicRecoveryWords(), expected)
	if err != nil {
		t.Fatal(err)
	}
	w, err := material.Wallet()
	if err != nil || w.GetAddress() != expected || len(w.ExportPrivateKeyHex()) != 64 {
		t.Fatal("legacy canonical address/key representation changed")
	}
	// Pasting words with whitespace must recover the canonical old wallet.
	pasted := "\n  " + strings.ReplaceAll(publicRecoveryWords(), " ", "  \t ") + "\n"
	if restored, e := RestoreRecoveryMaterial(pasted, expected); e != nil || restored.Address != expected {
		t.Fatal("normal word entry did not recover the old address")
	}
}

func TestMnemonicRecoveryCanRetainVerifiedOldRawSpacing(t *testing.T) {
	const expected = "14V5CxWM42z4AgJj9fHQ3Gy97oMpKLyYuN"
	words := strings.ReplaceAll(publicRecoveryWords(), " ", "  ")
	m, err := RestoreRecoveryMaterial(words, expected)
	if err != nil || m.Mnemonic != words || m.Address != expected {
		t.Fatal("old exact-input profile lost")
	}
	if _, err = m.Wallet(); err != nil {
		t.Fatal(err)
	}
}

func TestMnemonicRecoveryFailureNeverCreatesAnotherWallet(t *testing.T) {
	other, err := NewRecoveryMaterial()
	if err != nil {
		t.Fatal(err)
	}
	if result, err := RestoreRecoveryMaterial(publicRecoveryWords(), other.Address); result != nil || !errors.Is(err, ErrRecoveryAddressMismatch) {
		t.Fatal("unrelated phase-1 words were reported as a recovered wallet")
	}
	for _, words := range []string{"", strings.Repeat("abandon ", 12), strings.Repeat("x", 2049)} {
		if result, err := RestoreRecoveryMaterial(words, ""); err == nil || result != nil {
			t.Fatal("invalid words created a fallback wallet")
		}
	}
	if _, err := RestoreRecoveryMaterial(publicRecoveryWords(), "QCinvalid-legacy-demo"); err == nil {
		t.Fatal("invalid legacy display alias treated as a chain address")
	}
}

func TestNewMnemonicWalletHasFresh24WordRecovery(t *testing.T) {
	a, err := NewRecoveryMaterial()
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewRecoveryMaterial()
	if err != nil {
		t.Fatal(err)
	}
	if a.Mnemonic == b.Mnemonic || a.Address == b.Address || len(strings.Fields(a.Mnemonic)) != 24 || !ValidateAddress(a.Address) {
		t.Fatal("wallet generation did not use fresh mnemonic entropy")
	}
	if restored, err := RestoreRecoveryMaterial(a.Mnemonic, a.Address); err != nil || restored.Address != a.Address {
		t.Fatal("new-device word recovery changed the wallet")
	}
}

func TestEncryptedRecoveryVaultRoundtripAndTamperRejection(t *testing.T) {
	m, err := RestoreRecoveryMaterial(publicRecoveryWords(), "")
	if err != nil {
		t.Fatal(err)
	}
	password := "fixture-only-local-password"
	a, err := SealRecoveryMaterial(*m, password)
	if err != nil {
		t.Fatal(err)
	}
	b, err := SealRecoveryMaterial(*m, password)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a, b) || bytes.Contains(a, []byte(m.Mnemonic)) || bytes.Contains(a, []byte(password)) {
		t.Fatal("wallet encryption reused entropy or persisted plaintext")
	}
	got, err := OpenRecoveryMaterial(a, password)
	if err != nil || got.Address != m.Address || got.Mnemonic != m.Mnemonic {
		t.Fatal("encrypted recovery changed wallet")
	}
	if _, err := OpenRecoveryMaterial(a, "different-fixture-password"); err == nil {
		t.Fatal("wrong password accepted")
	}
	var v map[string]any
	if err := json.Unmarshal(a, &v); err != nil {
		t.Fatal(err)
	}
	for field, value := range map[string]any{"iterations": 1e12, "profile": "unknown", "nonce": "AA==", "address": "QCinvalid", "ciphertext": "AA==", "extra": "not allowed"} {
		copy := make(map[string]any)
		for k, val := range v {
			copy[k] = val
		}
		copy[field] = value
		raw, _ := json.Marshal(copy)
		if _, err := OpenRecoveryMaterial(raw, password); err == nil {
			t.Fatalf("tampered %s accepted", field)
		}
	}
	if _, err := OpenRecoveryMaterial(append(a, []byte("{}")...), password); err == nil {
		t.Fatal("trailing data accepted")
	}
}

func TestRecoveryVaultPersistenceNeverOverwritesExistingWallet(t *testing.T) {
	m, err := RestoreRecoveryMaterial(publicRecoveryWords(), "")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := SealRecoveryMaterial(*m, "fixture-only-local-password")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "wallet.qcvault")
	if err := SaveRecoveryVaultNew(path, raw); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := SaveRecoveryVaultNew(path, raw); err == nil {
		t.Fatal("existing wallet overwritten")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("old encrypted wallet changed")
	}
	if entries, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".qc-wallet-*")); len(entries) != 0 {
		t.Fatal("secret staging files leaked")
	}
}
