package blockchain

import (
	"bytes"
	"encoding/hex"
	"fmt"
	qconsensus "quantumcoin/consensus"
)

// BindPeerValidationContextFromLocalSynchronizedV1 initializes a decoded peer
// chain's private validation state from the canonical local node chain only.
// Peer payloads never supply authorities, program dates or activation policy.
// Blocks, UTXO, economic settings and ownership are not copied by this method;
// the strict local replacement gate remains authoritative for full admission.
func (bc *Blockchain) BindPeerValidationContextFromLocalSynchronizedV1(local *Blockchain) error {
	if bc == nil || local == nil {
		return ErrNilBlockchain
	}
	if bc == local {
		return fmt.Errorf("peer validation context: source and destination must differ")
	}
	local.stateMu.RLock()
	if len(local.Blocks) == 0 || local.Blocks[0] == nil {
		local.stateMu.RUnlock()
		return fmt.Errorf("peer validation context: local genesis missing")
	}
	genesis := append([]byte(nil), local.Blocks[0].Hash...)
	activation := local.activationPolicy
	trust := local.nftAwardTrust
	program := local.nftMonthlyProgram
	if trust.bound {
		trust.trust = cloneNFTAwardTrustV1(trust.trust)
	}
	local.stateMu.RUnlock()
	// An unbound local node resolves to LegacyV1, never a peer-selected plan.
	if !activation.bound {
		activation = activationPolicyBindingStateV1{bound: true}
	}
	if !trust.bound {
		trust = nftAwardTrustBindingStateV1{}
	}
	if !program.bound {
		program = nftMonthlyProgramBindingStateV1{}
	}
	if err := activation.policy.Validate(); err != nil {
		return fmt.Errorf("peer validation context: local activation: %w", err)
	}
	if activation.policy.Enabled {
		expected, err := hex.DecodeString(activation.policy.Plan.LegacyGenesisHex)
		if err != nil || !bytes.Equal(expected, genesis) {
			return fmt.Errorf("peer validation context: local activation genesis mismatch")
		}
	}
	if trust.bound {
		if err := qconsensus.ValidateNFTAwardTrustV1(trust.trust); err != nil {
			return err
		}
		expected, err := hex.DecodeString(trust.trust.LegacyGenesisHex)
		if err != nil || !bytes.Equal(expected, genesis) {
			return fmt.Errorf("peer validation context: local authority genesis mismatch")
		}
		if activation.policy.Enabled && activation.policy.Plan.NetworkID != trust.trust.NetworkID {
			return fmt.Errorf("peer validation context: local activation/authority networks differ")
		}
	}
	if program.bound {
		if err := program.policy.Validate(); err != nil {
			return err
		}
		if !trust.bound || program.policy.NetworkID != trust.trust.NetworkID {
			return fmt.Errorf("peer validation context: local program/authority networks differ")
		}
	}
	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()
	if bc.activationPolicy.bound || bc.nftAwardTrust.bound || bc.nftMonthlyProgram.bound {
		return fmt.Errorf("peer validation context: destination already bound")
	}
	if len(bc.Blocks) == 0 || bc.Blocks[0] == nil || !bytes.Equal(bc.Blocks[0].Hash, genesis) {
		return fmt.Errorf("peer validation context: incoming genesis differs from local chain")
	}
	bc.activationPolicy = activation
	bc.nftAwardTrust = trust
	bc.nftMonthlyProgram = program
	return nil
}
