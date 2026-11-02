package blockchain

import (
	"bytes"
	"encoding/hex"
	"fmt"

	qconsensus "quantumcoin/consensus"
)

// activationPolicyBindingStateV1 is private runtime state.
//
// Because the Blockchain field that holds this type is unexported,
// encoding/gob does not persist it inside chain files or peer payloads.
type activationPolicyBindingStateV1 struct {
	bound  bool
	policy qconsensus.ActivationPolicyV1
}

// BindActivationPolicyV1 validates and binds a local activation policy once.
//
// Peer data and serialized chain files cannot supply this runtime policy.
func (bc *Blockchain) BindActivationPolicyV1(
	policy qconsensus.ActivationPolicyV1,
) error {
	if bc == nil {
		return fmt.Errorf(
			"bind activation policy: nil blockchain",
		)
	}

	if bc.activationPolicy.bound {
		return fmt.Errorf(
			"activation policy is already bound",
		)
	}

	if err := policy.Validate(); err != nil {
		return fmt.Errorf(
			"bind activation policy: %w",
			err,
		)
	}

	if policy.Enabled {
		if len(bc.Blocks) == 0 ||
			bc.Blocks[0] == nil {
			return fmt.Errorf(
				"bind activation policy: blockchain has no genesis block",
			)
		}

		expectedGenesis, err := hex.DecodeString(
			policy.Plan.LegacyGenesisHex,
		)
		if err != nil {
			return fmt.Errorf(
				"bind activation policy: decode legacy genesis: %w",
				err,
			)
		}

		if !bytes.Equal(
			bc.Blocks[0].Hash,
			expectedGenesis,
		) {
			return fmt.Errorf(
				"bind activation policy: legacy genesis hash mismatch",
			)
		}
	}

	bc.activationPolicy = activationPolicyBindingStateV1{
		bound:  true,
		policy: policy,
	}

	return nil
}

// ProtocolAtHeightV1 returns LegacyV1 unless a validated local activation
// policy has been bound.
//
// Invalid or absent runtime state fails closed to LegacyV1.
func (bc *Blockchain) ProtocolAtHeightV1(
	height int,
) qconsensus.ProtocolVersion {
	if bc == nil ||
		!bc.activationPolicy.bound {
		return qconsensus.ProtocolLegacyV1
	}

	return bc.activationPolicy.policy.ProtocolAtHeight(
		height,
	)
}

// BindCanonicalActivationPolicyV1 binds the locally authoritative policy.
// The canonical policy currently remains safely disabled.
func (bc *Blockchain) BindCanonicalActivationPolicyV1() error {
	policy, err := qconsensus.LoadCanonicalActivationPolicyV1()
	if err != nil {
		return fmt.Errorf("canonical hardfork activation: %w", err)
	}
	return bc.BindActivationPolicyV1(policy)
}
