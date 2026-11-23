package blockchain

import "fmt"

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
		networkIdentity:         cloneNetworkIdentityV1(local.networkIdentity),
		StoredNetworkIdentityV1: cloneNetworkIdentityV1(local.StoredNetworkIdentityV1),
		Blocks:                  []*Block{cloneBlockV1(local.Blocks[0])},
		TotalSupply:             local.TotalSupply, coinbaseMaturity: local.coinbaseMaturity,
		activationPolicy: local.activationPolicy,
		nftAwardTrust:    local.nftAwardTrust, nftMonthlyProgram: local.nftMonthlyProgram,
	}
	if trusted.nftAwardTrust.bound {
		trusted.nftAwardTrust.trust = cloneNFTAwardTrustV1(trusted.nftAwardTrust.trust)
	}
	local.stateMu.RUnlock()
	loaded, err := loadDiskChainV1(filename, trusted)
	if err != nil {
		return nil, fmt.Errorf("trusted chain load: %w", err)
	}
	if loaded.TotalSupply != trusted.TotalSupply {
		return nil, fmt.Errorf("trusted chain load: total supply differs from local policy")
	}
	// Untagged legacy files may only be inspected/adopted outside mainnet.
	if trusted.networkIdentity != nil && loaded.StoredNetworkIdentityV1 == nil {
		if trusted.networkIdentity.Network == "mainnet" {
			return nil, fmt.Errorf("trusted chain load: unproven legacy network origin; mainnet refuses untagged chain")
		}
		loaded.StoredNetworkIdentityV1 = cloneNetworkIdentityV1(trusted.networkIdentity)
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
	return loaded, nil
}
