package blockchain

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	qconsensus "quantumcoin/consensus"
)

var ErrNFTAwardTrustUnboundV1 = errors.New("NFT award: local trust is not bound")

// The Blockchain field is unexported: gob cannot carry local authority policy.
// A loaded/peer chain needs explicit binding from trusted local bootstrap.
type nftAwardTrustBindingStateV1 struct {
	bound bool
	trust qconsensus.NFTAwardTrustV1
}

func cloneNFTAwardTrustV1(trust qconsensus.NFTAwardTrustV1) qconsensus.NFTAwardTrustV1 {
	result := trust
	result.AdminPublicKeys = make([][]byte, len(trust.AdminPublicKeys))
	for i, key := range trust.AdminPublicKeys {
		result.AdminPublicKeys[i] = append([]byte(nil), key...)
	}
	return result
}

// BindLocalNFTAwardTrustSynchronizedV1 binds a detached local authority policy
// exactly once. localNetworkID must come from local bootstrap configuration.
// This API must never be invoked with policy/network values supplied by peers.
// Policy shape validation is not a signature on a program release; no mint
// activation, campaign selection or eligibility decision is made here.
func (bc *Blockchain) BindLocalNFTAwardTrustSynchronizedV1(
	localNetworkID string,
	trust qconsensus.NFTAwardTrustV1,
) error {
	if bc == nil {
		return ErrNilBlockchain
	}
	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()
	if bc.nftAwardTrust.bound {
		return fmt.Errorf("NFT award: local trust is already bound")
	}
	policy := cloneNFTAwardTrustV1(trust)
	if err := qconsensus.ValidateNFTAwardTrustV1(policy); err != nil {
		return err
	}
	if localNetworkID != policy.NetworkID {
		return fmt.Errorf("NFT award: trust differs from local network")
	}
	if len(bc.Blocks) == 0 || bc.Blocks[0] == nil {
		return fmt.Errorf("NFT award: chain has no genesis")
	}
	genesis, err := hex.DecodeString(policy.LegacyGenesisHex)
	if err != nil || !bytes.Equal(genesis, bc.Blocks[0].Hash) {
		return fmt.Errorf("NFT award: trust differs from local genesis")
	}
	policy.LegacyGenesisHex = strings.ToLower(policy.LegacyGenesisHex)
	bc.nftAwardTrust = nftAwardTrustBindingStateV1{bound: true, trust: policy}
	return nil
}

// NFTAwardTrustBoundSnapshotV1 exposes status only, never mutable authority keys.
func (bc *Blockchain) NFTAwardTrustBoundSnapshotV1() bool {
	if bc == nil {
		return false
	}
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	return bc.nftAwardTrust.bound
}

// validateNFTAwardWithLocalTrustLockedV1 requires the caller's chain lock.
// No trust argument is accepted from the submitted award.
func (bc *Blockchain) validateNFTAwardWithLocalTrustLockedV1(
	record NFTMintRecordV1,
	decision qconsensus.NFTAwardDecisionV1,
	approvals []qconsensus.NFTAwardApprovalV1,
) error {
	if !bc.nftAwardTrust.bound {
		return ErrNFTAwardTrustUnboundV1
	}
	trust := bc.nftAwardTrust.trust
	if len(bc.Blocks) == 0 || bc.Blocks[0] == nil ||
		hex.EncodeToString(bc.Blocks[0].Hash) != trust.LegacyGenesisHex {
		return fmt.Errorf("NFT award: current chain genesis differs from bound trust")
	}
	return ValidateNFTAwardRecordApprovalV1(record, decision, trust, approvals)
}

// ValidateNFTAwardWithLocalTrustSynchronizedV1 verifies the submitted record
// using the chain's private, immutable local policy. Success is approval only,
// not block acceptance, minting, ownership or proof of Telegram eligibility.
func (bc *Blockchain) ValidateNFTAwardWithLocalTrustSynchronizedV1(
	record NFTMintRecordV1,
	decision qconsensus.NFTAwardDecisionV1,
	approvals []qconsensus.NFTAwardApprovalV1,
) error {
	if bc == nil {
		return ErrNilBlockchain
	}
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	return bc.validateNFTAwardWithLocalTrustLockedV1(record, decision, approvals)
}
