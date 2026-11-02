package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Manifest struct {
	Version       string `json:"version"`
	CreatedAt     string `json:"created_at"`
	WalletAddress string `json:"wallet_address"`
	PrivHexSHA256 string `json:"wallet_priv_hex_sha256"`
	WordsFile     string `json:"words_file"`
	Note          string `json:"note"`
}

func resolveWalletDir() (string, error) {
	walletDirFlag := flag.String(
		"wallet-dir",
		"",
		"directory containing recovery phase-1 files",
	)
	flag.Parse()

	walletDir := strings.TrimSpace(*walletDirFlag)
	if walletDir == "" {
		walletDir = strings.TrimSpace(os.Getenv("QC_WALLET_DIR"))
	}

	if walletDir == "" {
		return "", fmt.Errorf(
			"wallet directory required: use -wallet-dir <path> or QC_WALLET_DIR",
		)
	}

	absolutePath, err := filepath.Abs(filepath.Clean(walletDir))
	if err != nil {
		return "", fmt.Errorf("resolve wallet directory: %w", err)
	}

	info, err := os.Stat(absolutePath)
	if err != nil {
		return "", fmt.Errorf("wallet directory unavailable: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("wallet path is not a directory: %s", absolutePath)
	}

	return absolutePath, nil
}

func readRequired(path, label string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%s read failed: %w", label, err)
	}

	value := strings.TrimSpace(string(data))
	if value == "" {
		return "", fmt.Errorf("%s is empty: %s", label, path)
	}

	return value, nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "ERROR:", err)
	os.Exit(1)
}

func main() {
	walletDir, err := resolveWalletDir()
	if err != nil {
		fail(err)
	}

	wordsPath := filepath.Join(walletDir, "wallet_recovery_words.txt")
	privateKeyPath := filepath.Join(walletDir, "wallet_priv.hex")
	addressPath := filepath.Join(walletDir, "wallet_address.txt")
	manifestPath := filepath.Join(walletDir, "wallet_recovery_manifest.json")

	words, err := readRequired(wordsPath, "recovery words")
	if err != nil {
		fail(err)
	}

	privateKeyHex, err := readRequired(privateKeyPath, "wallet private key")
	if err != nil {
		fail(err)
	}

	address, err := readRequired(addressPath, "wallet address")
	if err != nil {
		fail(err)
	}

	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		fail(fmt.Errorf("read recovery manifest: %w", err))
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		fail(fmt.Errorf("decode recovery manifest: %w", err))
	}

	privateKeyHash := sha256.Sum256([]byte(privateKeyHex))
	currentHash := hex.EncodeToString(privateKeyHash[:])

	wordsPresent := strings.TrimSpace(words) != ""
	addressMatches := address == manifest.WalletAddress
	privateKeyMatches := currentHash == manifest.PrivHexSHA256

	fmt.Println("===================================")
	fmt.Println(" QuantumCoin Recovery Phase-1 Check")
	fmt.Println("===================================")
	fmt.Println("Wallet directory:", walletDir)
	fmt.Println("Wallet address:", address)
	fmt.Println("Manifest version:", manifest.Version)
	fmt.Println("Recovery words present:", wordsPresent)
	fmt.Println("Wallet address matches:", addressMatches)
	fmt.Println("Private-key hash matches:", privateKeyMatches)
	fmt.Println("Recovery words were not printed.")

	if !wordsPresent || !addressMatches || !privateKeyMatches {
		fmt.Println("STATUS: RECOVERY PHASE-1 CHECK FAILED")
		os.Exit(1)
	}

	fmt.Println("STATUS: RECOVERY PHASE-1 METADATA MATCH OK")
	fmt.Println("This does not yet prove deterministic wallet recovery.")
}
