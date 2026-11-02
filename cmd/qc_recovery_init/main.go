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
	"time"

	bip39 "github.com/tyler-smith/go-bip39"
)

type RecoveryManifest struct {
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
		"directory containing wallet_address.txt and wallet_priv.hex",
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
		return "", fmt.Errorf(
			"wallet path is not a directory: %s",
			absolutePath,
		)
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

func readOptional(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}

		return "", err
	}

	return strings.TrimSpace(string(data)), nil
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

	addressPath := filepath.Join(
		walletDir,
		"wallet_address.txt",
	)
	privateKeyPath := filepath.Join(
		walletDir,
		"wallet_priv.hex",
	)
	wordsPath := filepath.Join(
		walletDir,
		"wallet_recovery_words.txt",
	)
	manifestPath := filepath.Join(
		walletDir,
		"wallet_recovery_manifest.json",
	)

	address, err := readRequired(
		addressPath,
		"wallet address",
	)
	if err != nil {
		fail(err)
	}

	privateKeyHex, err := readRequired(
		privateKeyPath,
		"wallet private key",
	)
	if err != nil {
		fail(err)
	}

	words, err := readOptional(wordsPath)
	if err != nil {
		fail(fmt.Errorf("read recovery words: %w", err))
	}

	if words == "" {
		entropy, err := bip39.NewEntropy(128)
		if err != nil {
			fail(fmt.Errorf(
				"create recovery entropy: %w",
				err,
			))
		}

		words, err = bip39.NewMnemonic(entropy)
		if err != nil {
			fail(fmt.Errorf(
				"create recovery words: %w",
				err,
			))
		}

		err = os.WriteFile(
			wordsPath,
			[]byte(strings.TrimSpace(words)+"\n"),
			0600,
		)
		if err != nil {
			fail(fmt.Errorf(
				"write recovery words: %w",
				err,
			))
		}
	}

	privateKeyHash := sha256.Sum256(
		[]byte(privateKeyHex),
	)

	manifest := RecoveryManifest{
		Version:       "qc-recovery-phase1-v1",
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		WalletAddress: address,
		PrivHexSHA256: hex.EncodeToString(privateKeyHash[:]),
		WordsFile:     filepath.Base(wordsPath),
		Note: "Phase-1 metadata only. These words are linked " +
			"to the current wallet backup but cannot yet " +
			"recreate wallet_priv.hex.",
	}

	manifestBytes, err := json.MarshalIndent(
		manifest,
		"",
		"  ",
	)
	if err != nil {
		fail(fmt.Errorf(
			"encode recovery manifest: %w",
			err,
		))
	}

	manifestBytes = append(manifestBytes, byte('\n'))

	err = os.WriteFile(
		manifestPath,
		manifestBytes,
		0600,
	)
	if err != nil {
		fail(fmt.Errorf(
			"write recovery manifest: %w",
			err,
		))
	}

	fmt.Println("RECOVERY PHASE-1 METADATA READY")
	fmt.Println("Wallet directory:", walletDir)
	fmt.Println("Wallet address:", address)
	fmt.Println("Words file:", wordsPath)
	fmt.Println("Manifest:", manifestPath)
	fmt.Println("Recovery words were not printed.")
	fmt.Println("This is not yet deterministic wallet recovery.")
}
