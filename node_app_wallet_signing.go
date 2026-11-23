package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"quantumcoin/blockchain"
	"quantumcoin/utils"
	"quantumcoin/wallet"
	"strings"
)

// This command is a local wallet operation. It uses the executable's mandatory
// trusted chain bootstrap and does not submit, broadcast or save chain state.
func signWalletIntentV1(chain *blockchain.Blockchain, from, to, amount, key string, burn bool) (*blockchain.Transaction, error) {
	if !wallet.ValidateAddress(from) || (!burn && !wallet.ValidateAddress(to)) {
		return nil, fmt.Errorf("valid wallet addresses required")
	}
	var signer *wallet.Wallet
	var ok bool
	if key != "" {
		priv, err := wallet.ImportPrivateKeyHex(key)
		if err != nil {
			return nil, fmt.Errorf("invalid local signing key")
		}
		public, err := utils.SecpPublicBytesV1(&priv.PublicKey)
		if err != nil {
			return nil, fmt.Errorf("invalid local public key")
		}
		signer = &wallet.Wallet{PrivateKey: priv, PublicKey: public}
		ok = true
	} else {
		signer, ok = wallet.LoadWalletByAddress(from)
	}
	if !ok || signer == nil || signer.GetAddress() != from {
		return nil, fmt.Errorf("local key unavailable or does not belong to sender")
	}
	var tx *blockchain.Transaction
	var err error
	if burn {
		tx, err = blockchain.NewCanonicalBurnTransactionFromQCAmountV1(from, amount, chain)
	} else {
		tx, err = blockchain.NewTransactionFromQCAmountV1(from, to, amount, chain)
	}
	if err != nil {
		return nil, err
	}
	if err := chain.SignTransaction(tx, signer.PrivateKey); err != nil {
		return nil, fmt.Errorf("local transaction signing failed")
	}
	if !chain.VerifyTransaction(tx) {
		return nil, fmt.Errorf("local transaction verification failed")
	}
	return tx, nil
}

func runLocalSigningCommandV1(chain *blockchain.Blockchain, args []string) {
	burn := args[0] == "sign-burn"
	required := 4
	if burn {
		required = 3
	}
	if len(args) != required && len(args) != required+1 {
		fmt.Fprintln(os.Stderr, "usage: sign-send FROM TO AMOUNT [--key-stdin] or sign-burn FROM AMOUNT [--key-stdin]")
		return
	}
	key := ""
	if len(args) == required+1 {
		if args[required] != "--key-stdin" {
			fmt.Fprintln(os.Stderr, "unsupported signing option")
			return
		}
		data, err := io.ReadAll(io.LimitReader(os.Stdin, 1025))
		if err != nil || len(data) > 1024 {
			fmt.Fprintln(os.Stderr, "invalid signing input")
			return
		}
		key = strings.TrimSpace(string(data))
		if key == "" {
			fmt.Fprintln(os.Stderr, "signing key required on stdin")
			return
		}
	}
	from, to, amount := args[1], "", args[2]
	if !burn {
		to, amount = args[2], args[3]
	}
	tx, err := signWalletIntentV1(chain, from, to, amount, key, burn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "local signing failed:", err)
		return
	}
	data, err := json.Marshal(SendRequest{Transaction: tx})
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot encode signed transaction")
		return
	}
	fmt.Printf("SIGNED_TRANSACTION_V1=%s\n", data)
}
