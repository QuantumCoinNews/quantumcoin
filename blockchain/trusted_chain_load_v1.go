package blockchain

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
)

// LoadBlockchainFromFileWithLocalContextV1 validates a persisted chain against
// a coherent snapshot of a trusted local node's genesis and runtime policies.
// The local argument must never be a chain or policy supplied by a peer/file.
// Private activation/authorities/program/maturity are not decoded from gob;
// persisted UTXO and indexes are rebuilt only after full block validation.
// This does not enable a real program or choose startup authority keys/dates.
func LoadBlockchainFromFileWithLocalContextV1(filename string, local *Blockchain) (*Blockchain, error) {
	if local == nil {
		return nil, ErrNilBlockchain
	}
	local.stateMu.RLock()
	if len(local.Blocks) == 0 || local.Blocks[0] == nil {
		local.stateMu.RUnlock()
		return nil, fmt.Errorf("trusted chain load: local genesis is missing")
	}
	trusted := &Blockchain{
		Blocks:      []*Block{cloneBlockV1(local.Blocks[0])},
		TotalSupply: local.TotalSupply, coinbaseMaturity: local.coinbaseMaturity,
		activationPolicy: local.activationPolicy,
		nftAwardTrust:    local.nftAwardTrust, nftMonthlyProgram: local.nftMonthlyProgram,
	}
	if trusted.nftAwardTrust.bound {
		trusted.nftAwardTrust.trust = cloneNFTAwardTrustV1(trusted.nftAwardTrust.trust)
	}
	local.stateMu.RUnlock()
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("trusted chain load: read file: %w", err)
	}
	var loaded Blockchain
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&loaded); err != nil {
		return nil, fmt.Errorf("trusted chain load: decode file: %w", err)
	}
	if loaded.TotalSupply != trusted.TotalSupply {
		return nil, fmt.Errorf("trusted chain load: total supply differs from local policy")
	}
	if err := loaded.BindPeerValidationContextFromLocalSynchronizedV1(trusted); err != nil {
		return nil, fmt.Errorf("trusted chain load: bind local context: %w", err)
	}
	// Persistence may have equal height or contain genesis only. Reuse the
	// entire existing canonical replay, not peer chain-selection eligibility.
	if err := trusted.validateCanonicalChainWithLocalContextV1(loaded.Blocks); err != nil {
		return nil, fmt.Errorf("trusted chain load: canonical replay: %w", err)
	}
	loaded.coinbaseMaturity = trusted.coinbaseMaturity
	loaded.UTXO = map[string][]TransactionOutput{}
	loaded.pendingTxs = []*Transaction{}
	loaded.UpdateUTXOSet()
	loaded.rebuildTransactionLocationIndexesLocked()
	return &loaded, nil
}
