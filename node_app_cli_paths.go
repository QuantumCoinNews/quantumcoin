package main

import (
	"fmt"
	"os"
	"path/filepath"
	"quantumcoin/wallet"
	"strings"
)

type cliPathsV1 struct {
	Config, DataDir, Wallet string
	Args                    []string
}

// Selected once before services start. Zero values preserve caller-relative
// lookup for embedded/test adapters.
var activeCLIPathsV1 cliPathsV1

// Global path options precede the command. Resolution uses the caller's cwd,
// never os.Executable (go run places that binary in a temporary directory).
func parseCLIPathsV1(args []string) (cliPathsV1, error) {
	var p cliPathsV1
	walletFlag := false
	p.Config = strings.TrimSpace(os.Getenv("QC_CONFIG_FILE"))
	p.DataDir = strings.TrimSpace(os.Getenv("QC_DATA_DIR"))
	p.Wallet = strings.TrimSpace(os.Getenv("QC_WALLET_FILE"))
	for len(args) > 0 && strings.HasPrefix(args[0], "--") {
		if args[0] == "--help" {
			break
		}
		name, value, hasValue := strings.Cut(args[0], "=")
		if name != "--config" && name != "--data-dir" && name != "--wallet-file" {
			return p, fmt.Errorf("unknown option %q", name)
		}
		args = args[1:]
		if !hasValue {
			if len(args) == 0 || strings.HasPrefix(args[0], "--") {
				return p, fmt.Errorf("%s requires a path", name)
			}
			value, args = args[0], args[1:]
		}
		if strings.TrimSpace(value) == "" {
			return p, fmt.Errorf("%s requires a nonempty path", name)
		}
		switch name {
		case "--config":
			p.Config = value
		case "--data-dir":
			p.DataDir = value
		case "--wallet-file":
			p.Wallet = value
			walletFlag = true
		}
	}
	if walletFlag && (len(args) == 0 || (args[0] != "newaddr" && args[0] != "newaddr-priv")) {
		return p, fmt.Errorf("--wallet-file is for offline wallet creation; use QC_WALLET_FILE or wallet_file for node commands")
	}
	var err error
	if p.DataDir == "" {
		p.DataDir = "."
	}
	if p.DataDir, err = filepath.Abs(p.DataDir); err != nil {
		return p, err
	}
	if p.Config == "" {
		p.Config = filepath.Join(p.DataDir, "config.json")
	}
	if p.Config, err = filepath.Abs(p.Config); err != nil {
		return p, err
	}
	if p.Wallet == "" {
		p.Wallet = "wallet_data.json"
	}
	if !filepath.IsAbs(p.Wallet) {
		p.Wallet = filepath.Join(p.DataDir, p.Wallet)
	}
	p.Args = append([]string(nil), args...)
	return p, nil
}

// Validate dispatch and arity before any configuration/chain I/O.
func validateCLICommandV1(args []string) error {
	if len(args) == 0 {
		return nil
	}
	min, max := 1, 1
	switch args[0] {
	case "help", "--help", "-h", "api", "print", "newaddr", "newaddr-priv":
	case "run":
		max = 2
	case "run-mine", "mine", "mine-forever":
		min, max = 2, 2
	case "connect":
		min, max = 3, 3
	case "send":
		min, max = 4, 4
	case "sign-send":
		min, max = 4, 5
	case "sign-burn":
		min, max = 3, 4
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
	if len(args) < min || len(args) > max {
		return fmt.Errorf("invalid arguments for %s", args[0])
	}
	if (args[0] == "sign-send" || args[0] == "sign-burn") && len(args) == max && args[max-1] != "--key-stdin" {
		return fmt.Errorf("unsupported signing option")
	}
	return nil
}

func runOfflineCLICommandV1(p cliPathsV1) bool {
	if len(p.Args) == 0 {
		return false
	}
	switch p.Args[0] {
	case "help", "--help", "-h":
		printUsage()
		return true
	case "newaddr", "newaddr-priv":
		w := wallet.NewWallet()
		if err := wallet.SaveWalletAt(p.Wallet, w); err != nil {
			fmt.Fprintln(os.Stderr, "wallet persistence failed:", err)
			return true
		}
		fmt.Println("Wallet saved:", p.Wallet)
		fmt.Println("New Wallet Address:", w.GetAddress())
		if p.Args[0] == "newaddr-priv" {
			fmt.Println("PrivateKey (hex):", w.ExportPrivateKeyHex())
		}
		return true
	}
	return false
}
