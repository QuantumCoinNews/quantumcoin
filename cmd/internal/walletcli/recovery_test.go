package walletcli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"quantumcoin/wallet"
	"strings"
	"testing"
)

func TestRestoreEncryptedOnlyNeverOverwrites(t *testing.T) {
	words := strings.Repeat("abandon ", 11) + "about"
	password := "test-only-password-123456"
	input, _ := json.Marshal(map[string]string{"words": words, "password": password, "address": "1FLm5EcKtdBGNYg42Q21tfoXFwA5KYZy8W"})
	path := filepath.Join(t.TempDir(), "restored.qcvault")
	var out bytes.Buffer
	if err := Run("restore", path, bytes.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if bytes.Contains(raw, []byte("abandon")) || strings.Contains(out.String(), password) || strings.Contains(out.String(), words) {
		t.Fatal("plaintext secret exported")
	}
	m, err := wallet.OpenRecoveryMaterial(raw, password)
	if err != nil || m.Address != "1FLm5EcKtdBGNYg42Q21tfoXFwA5KYZy8W" {
		t.Fatal("legacy restore drift")
	}
	if Run("restore", path, bytes.NewReader(input), &out) == nil {
		t.Fatal("existing destination replaced")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(raw, after) {
		t.Fatal("existing encrypted file changed")
	}
}
