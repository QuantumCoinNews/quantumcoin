package main

import (
"crypto/sha256"
"encoding/hex"
"encoding/json"
"fmt"
"os"
"path/filepath"
"strings"
"time"

bip39 "github.com/tyler-smith/go-bip39"
)

type RecoveryManifest struct {
Version       string `json:"version"`
CreatedAt     string `json:"created_at"`
WalletAddress string `json:"wallet_address"`
PrivHexSHA256  string `json:"wallet_priv_hex_sha256"`
WordsFile     string `json:"words_file"`
Note          string `json:"note"`
}

func readTrim(path string) string {
b, err := os.ReadFile(path)
if err != nil {
return ""
}
return strings.TrimSpace(string(b))
}

func main() {
release := `C:\Projects\quantumcoin\release_clean_test`

addr := readTrim(filepath.Join(release, "wallet_address.txt"))
priv := readTrim(filepath.Join(release, "wallet_priv.hex"))

if addr == "" {
panic("wallet_address.txt not found or empty")
}
if priv == "" {
panic("wallet_priv.hex not found or empty")
}

wordsPath := filepath.Join(release, "wallet_recovery_words.txt")
manifestPath := filepath.Join(release, "wallet_recovery_manifest.json")

words := readTrim(wordsPath)
if words == "" {
entropy, err := bip39.NewEntropy(128)
if err != nil {
panic(err)
}
mnemonic, err := bip39.NewMnemonic(entropy)
if err != nil {
panic(err)
}
words = strings.TrimSpace(mnemonic)
if err := os.WriteFile(wordsPath, []byte(words+"\n"), 0600); err != nil {
panic(err)
}
}

sum := sha256.Sum256([]byte(priv))

manifest := RecoveryManifest{
Version:       "qc-recovery-v1",
CreatedAt:     time.Now().Format(time.RFC3339),
WalletAddress: addr,
PrivHexSHA256:  hex.EncodeToString(sum[:]),
WordsFile:     "wallet_recovery_words.txt",
Note:          "Phase 1 recovery words are linked to this wallet backup. Keep wallet_priv.hex and encrypted wallet backup safe until deterministic seed restore is implemented.",
}

b, _ := json.MarshalIndent(manifest, "", "  ")
if err := os.WriteFile(manifestPath, b, 0600); err != nil {
panic(err)
}

fmt.Println("RECOVERY WORDS READY")
fmt.Println("Address:", addr)
fmt.Println("Words :", words)
fmt.Println("Saved :", wordsPath)
fmt.Println("Manifest:", manifestPath)
}
