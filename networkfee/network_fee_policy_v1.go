// Package networkfee contains QuantumCoin project code for this module.
package networkfee

import (
	"errors"
	"strings"

	qconsensus "quantumcoin/consensus"
)

const NetworkFeePolicyVersionV1 = "QC_NETWORK_FEE_POLICY_V1"

const (
	MaxSupplyQCV1              int64  = qconsensus.CanonicalTotalSupplyQCV1
	DefaultMicroFeeTargetQC    string = "0.000001"
	DefaultCongestionCeilingQC string = "0.00001"

	DefaultMicroFeeTargetAtoms int64 = qconsensus.CanonicalAtomsPerQCV1 / 1_000_000

	DefaultCongestionCeilingAtoms int64  = qconsensus.CanonicalAtomsPerQCV1 / 100_000
	DefaultFeeRecipientV1         string = "BLOCK_PRODUCER_MINER"
	DefaultFeeModeV1              string = "MICRO_FEE_POLICY_FINAL_PARAMS_BEFORE_MAINNET"
)

type NetworkFeePolicy struct {
	Version                             string          `json:"version"`
	Mode                                string          `json:"mode"`
	MaxSupplyQC                         int64           `json:"maxSupplyQc"`
	BaseFeeTargetQC                     string          `json:"baseFeeTargetQc"`
	CongestionCeilingQC                 string          `json:"congestionCeilingQc"`
	FeeRecipient                        string          `json:"feeRecipient"`
	MiningActiveFeeDestination          string          `json:"miningActiveFeeDestination"`
	AfterEmissionDoneFeeDestination     string          `json:"afterEmissionDoneFeeDestination"`
	FeeIsNewMint                        bool            `json:"feeIsNewMint"`
	FeeChangesMaxSupply                 bool            `json:"feeChangesMaxSupply"`
	FeeGoesToFounder                    bool            `json:"feeGoesToFounder"`
	FeeGoesToBurnByDefault              bool            `json:"feeGoesToBurnByDefault"`
	FeeBurnOptionalFutureGovernanceOnly bool            `json:"feeBurnOptionalFutureGovernanceOnly"`
	FeeMustStayMicroLow                 bool            `json:"feeMustStayMicroLow"`
	FinalFeeParamsRequiredBeforeMainnet bool            `json:"finalFeeParamsRequiredBeforeMainnet"`
	AICannotChangeFee                   bool            `json:"aiCannotChangeFee"`
	AdminPanelCannotAutoChangeFee       bool            `json:"adminPanelCannotAutoChangeFee"`
	GameCannotChangeFee                 bool            `json:"gameCannotChangeFee"`
	WorkerCannotChangeFee               bool            `json:"workerCannotChangeFee"`
	NFTCannotChangeFee                  bool            `json:"nftCannotChangeFee"`
	TGWTCannotChangeFee                 bool            `json:"tgwtCannotChangeFee"`
	SafeBoundary                        map[string]bool `json:"safeBoundary"`
}

func DefaultPolicyV1() NetworkFeePolicy {
	return NetworkFeePolicy{
		Version:                             NetworkFeePolicyVersionV1,
		Mode:                                DefaultFeeModeV1,
		MaxSupplyQC:                         MaxSupplyQCV1,
		BaseFeeTargetQC:                     DefaultMicroFeeTargetQC,
		CongestionCeilingQC:                 DefaultCongestionCeilingQC,
		FeeRecipient:                        DefaultFeeRecipientV1,
		MiningActiveFeeDestination:          "BLOCK_REWARD_PLUS_TX_FEES_TO_BLOCK_PRODUCER",
		AfterEmissionDoneFeeDestination:     "TX_FEES_ONLY_TO_BLOCK_PRODUCER",
		FeeIsNewMint:                        false,
		FeeChangesMaxSupply:                 false,
		FeeGoesToFounder:                    false,
		FeeGoesToBurnByDefault:              false,
		FeeBurnOptionalFutureGovernanceOnly: true,
		FeeMustStayMicroLow:                 true,
		FinalFeeParamsRequiredBeforeMainnet: true,
		AICannotChangeFee:                   true,
		AdminPanelCannotAutoChangeFee:       true,
		GameCannotChangeFee:                 true,
		WorkerCannotChangeFee:               true,
		NFTCannotChangeFee:                  true,
		TGWTCannotChangeFee:                 true,
		SafeBoundary:                        SafeBoundaryV1(),
	}
}

func SafeBoundaryV1() map[string]bool {
	return map[string]bool{
		"doesNotMint":                         true,
		"doesNotChangeMaxSupply":              true,
		"doesNotChangeBalances":               true,
		"doesNotSendFunds":                    true,
		"doesNotExecuteFeeTransfer":           true,
		"doesNotChangeConsensus":              true,
		"doesNotChangeMiningRewardNow":        true,
		"feeIsNotFounderIncome":               true,
		"feeIsNotBurnByDefault":               true,
		"feeGoesToBlockProducer":              true,
		"feeMustStayMicroLow":                 true,
		"requiresFinalFeeParamsBeforeMainnet": true,
		"aiCannotChangeFee":                   true,
		"adminCannotAutoChangeFee":            true,
		"gameCannotChangeFee":                 true,
		"workerCannotChangeFee":               true,
		"controlOnly":                         true,
	}
}

func ValidatePolicyV1(policy NetworkFeePolicy) error {
	if strings.TrimSpace(policy.Version) != NetworkFeePolicyVersionV1 {
		return errors.New("invalid network fee policy version")
	}
	if strings.TrimSpace(policy.Mode) != DefaultFeeModeV1 {
		return errors.New("invalid network fee policy mode")
	}
	if policy.MaxSupplyQC != MaxSupplyQCV1 {
		return errors.New("fee policy must not change max supply")
	}
	if strings.TrimSpace(policy.BaseFeeTargetQC) != DefaultMicroFeeTargetQC {
		return errors.New("base fee target must be 0.000001 QC")
	}
	if strings.TrimSpace(policy.CongestionCeilingQC) != DefaultCongestionCeilingQC {
		return errors.New("congestion ceiling must be 0.00001 QC")
	}

	baseFeeAtoms, err := qconsensus.ParseQCToAtomsV1(
		policy.BaseFeeTargetQC,
	)
	if err != nil {
		return errors.New("base fee target has invalid QC denomination")
	}
	if baseFeeAtoms != DefaultMicroFeeTargetAtoms {
		return errors.New("base fee target must equal 1 atom")
	}

	congestionAtoms, err := qconsensus.ParseQCToAtomsV1(
		policy.CongestionCeilingQC,
	)
	if err != nil {
		return errors.New("congestion ceiling has invalid QC denomination")
	}
	if congestionAtoms != DefaultCongestionCeilingAtoms {
		return errors.New("congestion ceiling must equal 10 atoms")
	}
	if strings.TrimSpace(policy.FeeRecipient) != DefaultFeeRecipientV1 {
		return errors.New("fee recipient must be block producer/miner")
	}
	if strings.TrimSpace(policy.MiningActiveFeeDestination) !=
		"BLOCK_REWARD_PLUS_TX_FEES_TO_BLOCK_PRODUCER" {
		return errors.New("mining-active fee destination must be block producer")
	}
	if strings.TrimSpace(policy.AfterEmissionDoneFeeDestination) !=
		"TX_FEES_ONLY_TO_BLOCK_PRODUCER" {
		return errors.New("post-emission fee destination must be block producer")
	}
	if policy.FeeIsNewMint {
		return errors.New("fee must not be new mint")
	}
	if policy.FeeChangesMaxSupply {
		return errors.New("fee must not change max supply")
	}
	if policy.FeeGoesToFounder {
		return errors.New("fee must not go to founder wallet")
	}
	if policy.FeeGoesToBurnByDefault {
		return errors.New("fee must not burn by default")
	}
	if !policy.FeeMustStayMicroLow {
		return errors.New("fee must stay micro-low")
	}
	if !policy.FinalFeeParamsRequiredBeforeMainnet {
		return errors.New("final fee params required before mainnet")
	}
	if !policy.AICannotChangeFee || !policy.AdminPanelCannotAutoChangeFee || !policy.GameCannotChangeFee || !policy.WorkerCannotChangeFee {
		return errors.New("automation must not change fee")
	}
	if !policy.NFTCannotChangeFee || !policy.TGWTCannotChangeFee {
		return errors.New("NFT/TGWT must not change fee")
	}
	if err := validateExactSafeBoundaryMapV1(
		policy.SafeBoundary,
		SafeBoundaryV1(),
	); err != nil {
		return err
	}

	for _, k := range []string{
		"doesNotMint",
		"doesNotChangeMaxSupply",
		"doesNotChangeBalances",
		"doesNotSendFunds",
		"doesNotExecuteFeeTransfer",
		"doesNotChangeConsensus",
		"doesNotChangeMiningRewardNow",
		"feeIsNotFounderIncome",
		"feeIsNotBurnByDefault",
		"feeGoesToBlockProducer",
		"feeMustStayMicroLow",
		"requiresFinalFeeParamsBeforeMainnet",
		"aiCannotChangeFee",
		"adminCannotAutoChangeFee",
		"gameCannotChangeFee",
		"workerCannotChangeFee",
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
		return errors.New("network fee policy safe boundary required")
	}

	if len(boundary) != len(expected) {
		return errors.New("network fee policy safe boundary key count mismatch")
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
