package blockchain

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"

	"quantumcoin/consensus"
)

func networkSignatureDomainV2(domain string) bool {
	return domain == TestnetGenesisDomainV4 || domain == MainnetGenesisDomainV4
}

// Caller holds stateMu or owns a private validation scratch chain. The rule is
// intrinsic to genesis, never chosen by a transaction, an env flag or a peer.
func (bc *Blockchain) transactionGenesisLockedV2() (*[32]byte, error) {
	if bc == nil {
		return nil, ErrNilBlockchain
	}
	marker := ""
	if len(bc.Blocks) > 0 && bc.Blocks[0] != nil {
		genesis := bc.Blocks[0]
		if len(genesis.Transactions) > 0 && genesis.Transactions[0] != nil && len(genesis.Transactions[0].Outputs) > 0 {
			marker = string(genesis.Transactions[0].Outputs[0].PubKeyHash)
		}
	}
	if bc.networkIdentity != nil {
		if networkSignatureDomainV2(bc.networkIdentity.GenesisDomain) && marker != bc.networkIdentity.GenesisDomain {
			return nil, fmt.Errorf("transaction signing genesis domain changed")
		}
	}
	if !networkSignatureDomainV2(marker) {
		return nil, nil
	}
	genesis := bc.Blocks[0]
	if !isNetworkGenesisMarkerV1(genesis.Transactions[0]) || len(genesis.Hash) != 32 {
		return nil, fmt.Errorf("transaction signing requires a valid genesis")
	}
	var hash [32]byte
	copy(hash[:], genesis.Hash)
	if hash == ([32]byte{}) {
		return nil, fmt.Errorf("transaction signing requires a nonzero genesis")
	}
	if bc.networkIdentity != nil && (bc.networkIdentity.GenesisDomain != marker || bc.networkIdentity.GenesisHash != hex.EncodeToString(hash[:])) {
		return nil, fmt.Errorf("transaction signing genesis differs from trusted identity")
	}
	return &hash, nil
}

func transactionMessageForGenesisV2(tx *Transaction, index int, genesis *[32]byte) ([]byte, error) {
	if genesis == nil {
		return signMessageBytes(tx, index), nil
	}
	content, err := transactionInputDigestV2(tx, index)
	if err != nil {
		return nil, err
	}
	digest, err := consensus.NetworkSigningDigestV2(*genesis, content)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), digest[:]...), nil
}

// SignTransaction uses this chain's immutable signing rule. Legacy Sign is
// retained only for historical-format tools and byte-compatibility tests.
func (bc *Blockchain) SignTransaction(tx *Transaction, key *ecdsa.PrivateKey) error {
	if bc == nil {
		return ErrNilBlockchain
	}
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	genesis, err := bc.transactionGenesisLockedV2()
	if err != nil {
		return err
	}
	return tx.signForGenesisV2(key, genesis)
}

func (bc *Blockchain) verifyTransactionLockedV2(tx *Transaction) bool {
	genesis, err := bc.transactionGenesisLockedV2()
	return err == nil && tx.verifyForGenesisV2(genesis)
}

// VerifyTransaction is the shared chain-aware signature check. UTXO and amount
// validation remain authoritative in AddTransaction and block replay.
func (bc *Blockchain) VerifyTransaction(tx *Transaction) bool {
	if bc == nil {
		return false
	}
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	return bc.verifyTransactionLockedV2(tx)
}

// TransactionSigningHashes returns local trusted digests for external wallets.
func (bc *Blockchain) TransactionSigningHashes(tx *Transaction) ([]string, error) {
	if bc == nil || tx == nil {
		return nil, fmt.Errorf("chain and transaction required")
	}
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	genesis, err := bc.transactionGenesisLockedV2()
	if err != nil {
		return nil, err
	}
	result := make([]string, len(tx.Inputs))
	for i := range tx.Inputs {
		digest, err := transactionMessageForGenesisV2(tx, i, genesis)
		if err != nil {
			return nil, err
		}
		result[i] = hex.EncodeToString(digest)
	}
	return result, nil
}
