package main

import (
"bufio"
"crypto/ecdsa"
"crypto/sha256"
"encoding/hex"
"encoding/json"
"fmt"
"math/big"
"os"
"path/filepath"
"strings"
"time"

bip39 "github.com/tyler-smith/go-bip39"
"github.com/decred/dcrd/dcrec/secp256k1/v4"

"quantumcoin/utils"
"quantumcoin/wallet"
)

type Manifest struct {
Version       string `json:"version"`
CreatedAt     string `json:"created_at"`
WalletAddress string `json:"wallet_address"`
PrivHexSHA256 string `json:"wallet_priv_hex_sha256"`
WordsFile     string `json:"words_file"`
Note          string `json:"note"`
}

func walletFromWords(words string) (*wallet.Wallet, string) {
seed := bip39.NewSeed(words, "")
sum := sha256.Sum256(seed)

d := new(big.Int).SetBytes(sum[:])
curve := secp256k1.S256()
n := curve.Params().N

d.Mod(d, new(big.Int).Sub(n, big.NewInt(1)))
d.Add(d, big.NewInt(1))

priv := new(ecdsa.PrivateKey)
priv.PublicKey.Curve = curve
priv.D = d
priv.PublicKey.X, priv.PublicKey.Y = curve.ScalarBaseMult(d.Bytes())

w := &wallet.Wallet{
PrivateKey: priv,
PublicKey:  utils.MakeUncompressedPub(priv),
}

return w, hex.EncodeToString(d.Bytes())
}

func main() {
release := `C:\Projects\quantumcoin\release`

fmt.Println("===================================")
fmt.Println(" QuantumCoin Canonical Wallet Restore")
fmt.Println("===================================")
fmt.Println()

fmt.Print("Enter recovery words: ")

reader := bufio.NewReader(os.Stdin)
words, _ := reader.ReadString('\n')
words = strings.TrimSpace(words)

if !bip39.IsMnemonicValid(words) {
fmt.Println()
fmt.Println("ERROR: INVALID RECOVERY WORDS")
return
}

w, privHex := walletFromWords(words)
addr := w.GetAddress()

_ = os.WriteFile(filepath.Join(release, "wallet_recovery_words.txt"), []byte(words), 0644)
_ = os.WriteFile(filepath.Join(release, "wallet_priv.hex"), []byte(privHex), 0644)
_ = os.WriteFile(filepath.Join(release, "wallet_address.txt"), []byte(addr), 0644)
_ = os.WriteFile(filepath.Join(release, "miner_address.txt"), []byte(addr), 0644)

privHash := sha256.Sum256([]byte(privHex))

manifest := Manifest{
Version:       "qc-canonical-deterministic-wallet-restore-v1",
CreatedAt:     time.Now().Format(time.RFC3339),
WalletAddress: addr,
PrivHexSHA256: hex.EncodeToString(privHash[:]),
WordsFile:     "wallet_recovery_words.txt",
Note:          "Canonical QuantumCoin Base58 wallet restored from BIP39 recovery words.",
}

b, _ := json.MarshalIndent(manifest, "", "  ")
_ = os.WriteFile(filepath.Join(release, "wallet_recovery_manifest.json"), b, 0644)

fmt.Println()
fmt.Println("WALLET RESTORED")
fmt.Println()
fmt.Println("RESTORED ADDRESS:")
fmt.Println(addr)
fmt.Println()
fmt.Println("FILES WRITTEN TO:")
fmt.Println(release)
}
