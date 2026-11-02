// Package economycontrol contains QuantumCoin project code for this module.
package economycontrol

import (
	"errors"
	"strings"

	qconsensus "quantumcoin/consensus"
)

const MainnetBeforeEconomyControlVersionV1 = "QC_MAINNET_BEFORE_ECONOMY_CONTROL_V1"

const (
	TotalSupplyQCV1             int64 = qconsensus.CanonicalTotalSupplyQCV1
	FounderTotalBasisPointsV1   int   = qconsensus.CanonicalFounderTotalBasisPointsV1
	FounderLiquidBasisPointsV1  int   = qconsensus.CanonicalFounderLiquidBasisPointsV1
	FounderLockedBasisPointsV1  int   = qconsensus.CanonicalFounderLockedBasisPointsV1
	NonFounderBasisPointsV1     int   = qconsensus.CanonicalNonFounderBasisPointsV1
	FullAllocationBasisPointsV1 int   = qconsensus.CanonicalFullAllocationBasisPointsV1
	InitialMiningRewardQCV1     int64 = qconsensus.CanonicalInitialMiningRewardQCV1
	CoinbaseMaturityBlocksV1    int64 = qconsensus.CanonicalCoinbaseMaturityBlocksV1
)

type Allocation struct {
	Name        string `json:"name"`
	BasisPoints int    `json:"basisPoints"`
	Locked      bool   `json:"locked"`
	Liquid      bool   `json:"liquid"`
	MintsNewQC  bool   `json:"mintsNewQc"`
}

type EconomyPolicy struct {
	Version                                string            `json:"version"`
	Mode                                   string            `json:"mode"`
	TotalSupplyQC                          int64             `json:"totalSupplyQc"`
	FounderTotalBasisPoints                int               `json:"founderTotalBasisPoints"`
	FounderLiquidBasisPoints               int               `json:"founderLiquidBasisPoints"`
	FounderLockedBasisPoints               int               `json:"founderLockedBasisPoints"`
	FounderWalletBindingRequired           bool              `json:"founderWalletBindingRequired"`
	FounderWalletAddressFinalBeforeMainnet bool              `json:"founderWalletAddressFinalBeforeMainnet"`
	InitialMiningRewardQC                  int64             `json:"initialMiningRewardQc"`
	CoinbaseMaturityBlocks                 int64             `json:"coinbaseMaturityBlocks"`
	Allocations                            []Allocation      `json:"allocations"`
	SafeBoundary                           map[string]bool   `json:"safeBoundary"`
	Notes                                  map[string]string `json:"notes"`
}

func DefaultPolicyV1() EconomyPolicy {
	return EconomyPolicy{
		Version:                                MainnetBeforeEconomyControlVersionV1,
		Mode:                                   "PRE_MAINNET_ECONOMY_CONTROL_NO_LAUNCH",
		TotalSupplyQC:                          TotalSupplyQCV1,
		FounderTotalBasisPoints:                FounderTotalBasisPointsV1,
		FounderLiquidBasisPoints:               FounderLiquidBasisPointsV1,
		FounderLockedBasisPoints:               FounderLockedBasisPointsV1,
		FounderWalletBindingRequired:           true,
		FounderWalletAddressFinalBeforeMainnet: true,
		InitialMiningRewardQC:                  InitialMiningRewardQCV1,
		CoinbaseMaturityBlocks:                 CoinbaseMaturityBlocksV1,
		Allocations: []Allocation{
			{
				Name:        "founder_liquid_operations",
				BasisPoints: FounderLiquidBasisPointsV1,
				Locked:      false,
				Liquid:      true,
				MintsNewQC:  false,
			},
			{
				Name:        "founder_locked_vesting",
				BasisPoints: FounderLockedBasisPointsV1,
				Locked:      true,
				Liquid:      false,
				MintsNewQC:  false,
			},
			{
				Name:        "non_founder_network_supply",
				BasisPoints: NonFounderBasisPointsV1,
				Locked:      false,
				Liquid:      false,
				MintsNewQC:  false,
			},
		},
		SafeBoundary: SafeBoundaryV1(),
		Notes: map[string]string{
			"status":  "Pre-mainnet economy control only. This does not launch mainnet.",
			"supply":  "Total supply is fixed at 25,500,000 QC.",
			"founder": "Founder/main allocation is 5% total: 1% liquid and 4% locked/vesting. Final founder wallet address binding is required before mainnet.",
			"ai":      "AI can monitor, score, and report only. AI cannot mint, change balances, change consensus, or alter supply.",
			"tgwt":    "TGWT is off-chain/game ecosystem accounting and cannot mint QC or change QC supply.",
			"nft":     "NFT review/mint flow cannot change QC supply, QC balances, mining rewards, or consensus.",
			"testnet": "Testnet may test logic but cannot override mainnet economic invariants.",
			"mainnet": "Any mainnet economy change requires explicit final review before launch.",
		},
	}
}

func SafeBoundaryV1() map[string]bool {
	return map[string]bool{
		"doesNotLaunchMainnet":              true,
		"doesNotMint":                       true,
		"doesNotSendFunds":                  true,
		"doesNotChangeBalances":             true,
		"doesNotChangeChain":                true,
		"doesNotChangeConsensus":            true,
		"doesNotChangeSupply":               true,
		"doesNotChangeMiningReward":         true,
		"doesNotStartMining":                true,
		"doesNotStopMining":                 true,
		"doesNotSelectNFTWinners":           true,
		"doesNotExecuteSettlement":          true,
		"preMainnetControlOnly":             true,
		"requiresFinalHumanApproval":        true,
		"requiresImmutableSupplyCheck":      true,
		"requiresFounderSplitCheck":         true,
		"requiresFounderWalletBinding":      true,
		"requiresFinalFounderWalletAddress": true,
	}
}

func ValidatePolicyV1(policy EconomyPolicy) error {
	if strings.TrimSpace(policy.Version) != MainnetBeforeEconomyControlVersionV1 {
		return errors.New("invalid economy control version")
	}
	if strings.TrimSpace(policy.Mode) != "PRE_MAINNET_ECONOMY_CONTROL_NO_LAUNCH" {
		return errors.New("invalid economy control mode")
	}
	if policy.TotalSupplyQC != TotalSupplyQCV1 {
		return errors.New("invalid total supply")
	}
	if policy.FounderTotalBasisPoints != FounderTotalBasisPointsV1 {
		return errors.New("invalid founder total basis points")
	}
	if policy.FounderLiquidBasisPoints != FounderLiquidBasisPointsV1 {
		return errors.New("invalid founder liquid basis points")
	}
	if policy.FounderLockedBasisPoints != FounderLockedBasisPointsV1 {
		return errors.New("invalid founder locked basis points")
	}
	if policy.FounderLiquidBasisPoints+policy.FounderLockedBasisPoints != policy.FounderTotalBasisPoints {
		return errors.New("founder liquid + locked must equal founder total")
	}
	if !policy.FounderWalletBindingRequired {
		return errors.New("founder wallet binding must be required")
	}
	if !policy.FounderWalletAddressFinalBeforeMainnet {
		return errors.New("founder wallet address must be finalized before mainnet")
	}
	if policy.InitialMiningRewardQC != InitialMiningRewardQCV1 {
		return errors.New("invalid initial mining reward")
	}
	if policy.CoinbaseMaturityBlocks != CoinbaseMaturityBlocksV1 {
		return errors.New("invalid coinbase maturity")
	}
	if len(policy.Allocations) < 1 {
		return errors.New("allocations required")
	}

	total := 0
	seen := map[string]bool{}
	for _, a := range policy.Allocations {
		name := strings.TrimSpace(a.Name)
		if name == "" {
			return errors.New("allocation name required")
		}
		if seen[name] {
			return errors.New("duplicate allocation: " + name)
		}
		seen[name] = true

		switch name {
		case "founder_liquid_operations":
			if a.BasisPoints != FounderLiquidBasisPointsV1 || a.Locked || !a.Liquid {
				return errors.New("invalid founder liquid allocation")
			}
		case "founder_locked_vesting":
			if a.BasisPoints != FounderLockedBasisPointsV1 || !a.Locked || a.Liquid {
				return errors.New("invalid founder locked allocation")
			}
		case "non_founder_network_supply":
			if a.BasisPoints != NonFounderBasisPointsV1 || a.Locked || a.Liquid {
				return errors.New("invalid non-founder network allocation")
			}
		}
		if a.BasisPoints <= 0 {
			return errors.New("allocation basis points must be positive: " + name)
		}
		if a.MintsNewQC {
			return errors.New("allocation must not mint new QC: " + name)
		}
		total += a.BasisPoints
	}
	if total != FullAllocationBasisPointsV1 {
		return errors.New("allocation basis points must equal 10000")
	}

	for _, name := range []string{
		"founder_liquid_operations",
		"founder_locked_vesting",
		"non_founder_network_supply",
	} {
		if !seen[name] {
			return errors.New("missing required allocation: " + name)
		}
	}
	if err := validateExactSafeBoundaryMapV1(
		policy.SafeBoundary,
		SafeBoundaryV1(),
	); err != nil {
		return err
	}

	for _, k := range []string{
		"doesNotLaunchMainnet",
		"doesNotMint",
		"doesNotSendFunds",
		"doesNotChangeBalances",
		"doesNotChangeChain",
		"doesNotChangeConsensus",
		"doesNotChangeSupply",
		"doesNotChangeMiningReward",
		"doesNotStartMining",
		"doesNotStopMining",
		"doesNotSelectNFTWinners",
		"doesNotExecuteSettlement",
		"preMainnetControlOnly",
		"requiresFinalHumanApproval",
		"requiresImmutableSupplyCheck",
		"requiresFounderSplitCheck",
		"requiresFounderWalletBinding",
		"requiresFinalFounderWalletAddress",
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
		return errors.New("economy policy safe boundary required")
	}

	if len(boundary) != len(expected) {
		return errors.New("economy policy safe boundary key count mismatch")
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
