package main

import (
"crypto/sha256"
"encoding/hex"
"encoding/json"
"fmt"
"os"
"path/filepath"
"strings"
)

type Manifest struct {
Version           string `json:"version"`
CreatedAt         string `json:"created_at"`
WalletAddress     string `json:"wallet_address"`
PrivHexSHA256     string `json:"wallet_priv_hex_sha256"`
WordsFile         string `json:"words_file"`
Note              string `json:"note"`
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

words := readTrim(filepath.Join(release, "wallet_recovery_words.txt"))
priv := readTrim(filepath.Join(release, "wallet_priv.hex"))
addr := readTrim(filepath.Join(release, "wallet_address.txt"))

var mf Manifest

manifestBytes, err := os.ReadFile(filepath.Join(release, "wallet_recovery_manifest.json"))
if err == nil {
_ = json.Unmarshal(manifestBytes, &mf)
}

sum := sha256.Sum256([]byte(priv))
hash := hex.EncodeToString(sum[:])

fmt.Println("===================================")
fmt.Println(" QuantumCoin Seed Manager")
fmt.Println("===================================")
fmt.Println()

fmt.Println("Wallet Address:")
fmt.Println(addr)
fmt.Println()

fmt.Println("Recovery Words:")
fmt.Println(words)
fmt.Println()

fmt.Println("Manifest Hash:")
fmt.Println(mf.PrivHexSHA256)
fmt.Println()

fmt.Println("Current Wallet Hash:")
fmt.Println(hash)
fmt.Println()

if hash == mf.PrivHexSHA256 {
fmt.Println("STATUS: WALLET MATCH OK")
} else {
fmt.Println("STATUS: WALLET HASH MISMATCH")
}

fmt.Println()
fmt.Println("Recovery system phase-1 ready.")
}
