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

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	bip39 "github.com/tyler-smith/go-bip39"

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

func failCLI(format string, args ...interface{}) {
	fmt.Fprintf(
		os.Stderr,
		"ERROR: "+format+"\n",
		args...,
	)
	os.Exit(1)
}

func writeRequiredCLIFile(path string, data []byte) {
	if err := os.WriteFile(path, data, 0644); err != nil {
		failCLI(
			"write %s: %v",
			filepath.Base(path),
			err,
		)
	}
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

	writeRequiredCLIFile(filepath.Join(release, "wallet_recovery_words.txt"), []byte(words))
	writeRequiredCLIFile(filepath.Join(release, "wallet_priv.hex"), []byte(privHex))
	writeRequiredCLIFile(filepath.Join(release, "wallet_address.txt"), []byte(addr))
	writeRequiredCLIFile(filepath.Join(release, "miner_address.txt"), []byte(addr))

	privHash := sha256.Sum256([]byte(privHex))

	manifest := Manifest{
		Version:       "qc-canonical-deterministic-wallet-restore-v1",
		CreatedAt:     time.Now().Format(time.RFC3339),
		WalletAddress: addr,
		PrivHexSHA256: hex.EncodeToString(privHash[:]),
		WordsFile:     "wallet_recovery_words.txt",
		Note:          "Canonical QuantumCoin Base58 wallet restored from BIP39 recovery words.",
	}

	b, marshalErr := json.MarshalIndent(manifest, "", "  ")
	if marshalErr != nil {
		failCLI(
			"encode wallet recovery manifest: %v",
			marshalErr,
		)
	}
	writeRequiredCLIFile(
		filepath.Join(release, "wallet_recovery_manifest.json"),
		b,
	)

	fmt.Println()
	fmt.Println("WALLET RESTORED")
	fmt.Println()
	fmt.Println("RESTORED ADDRESS:")
	fmt.Println(addr)
	fmt.Println()
	fmt.Println("FILES WRITTEN TO:")
	fmt.Println(release)
}
