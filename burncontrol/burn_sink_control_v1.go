// Package burncontrol contains QuantumCoin project code for this module.
package burncontrol

import (
	"errors"
	"strings"

	qconsensus "quantumcoin/consensus"
)

const BurnSinkControlVersionV1 = "QC_BURN_SINK_CONTROL_V1"

const (
	MaxSupplyQCV1        int64  = qconsensus.CanonicalTotalSupplyQCV1
	DefaultBurnAddressV1 string = qconsensus.CanonicalBurnSinkAddressV1
)

type BurnPolicy struct {
	Version                              string          `json:"version"`
	Mode                                 string          `json:"mode"`
	MaxSupplyQC                          int64           `json:"maxSupplyQc"`
	BurnAddressAlias                     string          `json:"burnAddressAlias"`
	BurnAddressFinalBeforeMainnet        bool            `json:"burnAddressFinalBeforeMainnet"`
	BurnAddressMustBeUnspendable         bool            `json:"burnAddressMustBeUnspendable"`
	BurnPrivateKeyMustNotExist           bool            `json:"burnPrivateKeyMustNotExist"`
	BurnedSupplyReducesCirculatingSupply bool            `json:"burnedSupplyReducesCirculatingSupply"`
	BurnDoesNotReduceMaxSupply           bool            `json:"burnDoesNotReduceMaxSupply"`
	BurnDoesNotMintReplacement           bool            `json:"burnDoesNotMintReplacement"`
	AutoBurnForbidden                    bool            `json:"autoBurnForbidden"`
	RequiresExplicitApproval             bool            `json:"requiresExplicitApproval"`
	AllowedBurnSources                   []string        `json:"allowedBurnSources"`
	SafeBoundary                         map[string]bool `json:"safeBoundary"`
}

func DefaultPolicyV1() BurnPolicy {
	return BurnPolicy{
		Version:                              BurnSinkControlVersionV1,
		Mode:                                 "BURN_SINK_CONTROL_NO_BURN_EXECUTION",
		MaxSupplyQC:                          MaxSupplyQCV1,
		BurnAddressAlias:                     DefaultBurnAddressV1,
		BurnAddressFinalBeforeMainnet:        true,
		BurnAddressMustBeUnspendable:         true,
		BurnPrivateKeyMustNotExist:           true,
		BurnedSupplyReducesCirculatingSupply: true,
		BurnDoesNotReduceMaxSupply:           true,
		BurnDoesNotMintReplacement:           true,
		AutoBurnForbidden:                    true,
		RequiresExplicitApproval:             true,
		AllowedBurnSources: []string{
			"explicit_user_burn",
			"explicit_admin_approved_burn",
		},
		SafeBoundary: SafeBoundaryV1(),
	}
}

func SafeBoundaryV1() map[string]bool {
	return map[string]bool{
		"doesNotExecuteBurn":                    true,
		"doesNotSendFunds":                      true,
		"doesNotMint":                           true,
		"doesNotMintReplacement":                true,
		"doesNotChangeBalances":                 true,
		"doesNotChangeChain":                    true,
		"doesNotChangeConsensus":                true,
		"doesNotChangeMaxSupply":                true,
		"doesNotChangeMiningReward":             true,
		"doesNotStartMining":                    true,
		"doesNotStopMining":                     true,
		"requiresBurnAddressFinalBeforeMainnet": true,
		"requiresUnspendableBurnAddress":        true,
		"requiresNoBurnPrivateKey":              true,
		"requiresExplicitBurnApproval":          true,
		"burnReducesCirculatingSupplyOnly":      true,
		"controlOnly":                           true,
	}
}

func ValidatePolicyV1(policy BurnPolicy) error {
	if strings.TrimSpace(policy.Version) != BurnSinkControlVersionV1 {
		return errors.New("invalid burn sink control version")
	}
	if strings.TrimSpace(policy.Mode) != "BURN_SINK_CONTROL_NO_BURN_EXECUTION" {
		return errors.New("invalid burn sink control mode")
	}
	if policy.MaxSupplyQC != MaxSupplyQCV1 {
		return errors.New("burn policy must not change max supply")
	}
	if strings.TrimSpace(policy.BurnAddressAlias) == "" {
		return errors.New("burn address alias required")
	}
	if strings.TrimSpace(policy.BurnAddressAlias) != DefaultBurnAddressV1 {
		return errors.New("burn address alias must match default burn sink")
	}
	if !policy.BurnAddressFinalBeforeMainnet {
		return errors.New("burn address must be finalized before mainnet")
	}
	if !policy.BurnAddressMustBeUnspendable {
		return errors.New("burn address must be unspendable")
	}
	if !policy.BurnPrivateKeyMustNotExist {
		return errors.New("burn private key must not exist")
	}
	if !policy.BurnedSupplyReducesCirculatingSupply {
		return errors.New("burned supply must reduce circulating supply")
	}
	if !policy.BurnDoesNotReduceMaxSupply {
		return errors.New("burn must not reduce max supply constant")
	}
	if !policy.BurnDoesNotMintReplacement {
		return errors.New("burn must not mint replacement QC")
	}
	if !policy.AutoBurnForbidden {
		return errors.New("auto burn must be forbidden")
	}
	if !policy.RequiresExplicitApproval {
		return errors.New("explicit burn approval required")
	}
	if len(policy.AllowedBurnSources) < 1 {
		return errors.New("allowed burn sources required")
	}
	if err := validateExactSafeBoundaryMapV1(
		policy.SafeBoundary,
		SafeBoundaryV1(),
	); err != nil {
		return err
	}

	for _, k := range []string{
		"doesNotExecuteBurn",
		"doesNotSendFunds",
		"doesNotMint",
		"doesNotMintReplacement",
		"doesNotChangeBalances",
		"doesNotChangeChain",
		"doesNotChangeConsensus",
		"doesNotChangeMaxSupply",
		"doesNotChangeMiningReward",
		"doesNotStartMining",
		"doesNotStopMining",
		"requiresBurnAddressFinalBeforeMainnet",
		"requiresUnspendableBurnAddress",
		"requiresNoBurnPrivateKey",
		"requiresExplicitBurnApproval",
		"burnReducesCirculatingSupplyOnly",
		"controlOnly",
	} {
		if !policy.SafeBoundary[k] {
			return errors.New("safe boundary missing: " + k)
		}
	}

	return nil
}

func validateExactSafeBoundaryMapV1(
	boundary map[string]bool,
	expected map[string]bool,
) error {
	if boundary == nil {
		return errors.New("burn policy safe boundary required")
	}

	if len(boundary) != len(expected) {
		return errors.New("burn policy safe boundary key count mismatch")
	}

	for key, want := range expected {
		got, exists := boundary[key]
		if !exists {
			return errors.New("safe boundary missing: " + key)
		}
		if got != want {
			return errors.New("safe boundary value mismatch: " + key)
		}
	}

	for key := range boundary {
		if _, exists := expected[key]; !exists {
			return errors.New("unexpected safe boundary key: " + key)
		}
	}

	return nil
}
