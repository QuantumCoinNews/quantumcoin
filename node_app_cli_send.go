package main

import (
	"fmt"
	"strings"

	"quantumcoin/blockchain"
	"quantumcoin/wallet"
)

// submitCLISendV1 uses only the requested sender's existing wallet. No default
// wallet fallback, key generation or private-key command-line argument is used.
// Amount conversion and admission share the API's canonical boundaries.
func submitCLISendV1(chain *blockchain.Blockchain, from, to, amount string) (*blockchain.Transaction, error) {
	from, to = strings.TrimSpace(from), strings.TrimSpace(to)
	if !wallet.ValidateAddress(from) || !wallet.ValidateAddress(to) {
		return nil, fmt.Errorf("valid from and to wallet addresses required")
	}
	if chain == nil {
		return nil, fmt.Errorf("blockchain unavailable")
	}
	signer, ok := wallet.LoadWalletByAddress(from)
	if !ok || signer == nil || signer.PrivateKey == nil {
		return nil, fmt.Errorf("sender private key unavailable in configured wallet store")
	}
	if signer.GetAddress() != from {
		return nil, fmt.Errorf("stored private key does not belong to sender")
	}
	tx, err := blockchain.NewTransactionFromQCAmountV1(from, to, strings.TrimSpace(amount), chain)
	if err != nil {
		return nil, fmt.Errorf("create transaction: %w", err)
	}
	if err := chain.SignTransaction(tx, signer.PrivateKey); err != nil {
		return nil, fmt.Errorf("sign transaction: %w", err)
	}
	if !chain.VerifyTransaction(tx) {
		return nil, fmt.Errorf("signed transaction verification failed")
	}
	if err := chain.AddTransaction(tx); err != nil {
		return nil, fmt.Errorf("submit transaction: %w", err)
	}
	return tx, nil
}
