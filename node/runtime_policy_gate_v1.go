package node

import (
	"fmt"
	"strings"

	"quantumcoin/blockchain"
	"quantumcoin/burncontrol"
	"quantumcoin/config"
	"quantumcoin/consensus"
	"quantumcoin/economycontrol"
	"quantumcoin/foundervesting"
	"quantumcoin/hardforkpolicy"
	"quantumcoin/networkfee"
)

const RuntimePolicyGateVersionV1 = "QC_RUNTIME_POLICY_GATE_V1"

// RuntimePolicySetV1 contains the declarative policies that must agree with
// the canonical config, consensus and blockchain implementations before node
// services are allowed to start.
type RuntimePolicySetV1 struct {
	Burn           burncontrol.BurnPolicy
	Economy        economycontrol.EconomyPolicy
	FounderVesting foundervesting.FounderVestingPolicy
	Hardfork       hardforkpolicy.HardforkPolicy
	NetworkFee     networkfee.NetworkFeePolicy
}

// DefaultRuntimePolicySetV1 returns a fresh policy set. Callers cannot mutate
// shared process-global policy state.
func DefaultRuntimePolicySetV1() RuntimePolicySetV1 {
	return RuntimePolicySetV1{
		Burn:           burncontrol.DefaultPolicyV1(),
		Economy:        economycontrol.DefaultPolicyV1(),
		FounderVesting: foundervesting.DefaultPolicyV1(),
		Hardfork:       hardforkpolicy.DefaultPolicyV1(),
		NetworkFee:     networkfee.DefaultPolicyV1(),
	}
}

// ValidateRuntimePolicyGateV1 is the fail-closed startup boundary connecting
// declarative policy packages to the canonical runtime implementations.
//
// It validates each policy, verifies that duplicated declarations agree with
// canonical config/consensus/blockchain values and prevents explicit mainnet
// startup while declared-but-unimplemented burn-sink and fractional-fee
// requirements remain unresolved.
func ValidateRuntimePolicyGateV1(
	cfg *config.Config,
	policies RuntimePolicySetV1,
) error {
	release, snapshotDir, err := consensus.LoadCanonicalHardforkReleaseV1()
	if err != nil {
		return err
	}
	activation, err := consensus.LoadHardforkActivationV1(release, snapshotDir)
	if err != nil {
		return fmt.Errorf("%s: hardfork activation: %w", RuntimePolicyGateVersionV1, err)
	}
	return validateRuntimePolicyGateWithActivationV1(cfg, policies, activation)
}

func validateRuntimePolicyGateWithActivationV1(
	cfg *config.Config,
	policies RuntimePolicySetV1,
	activation consensus.ActivationPolicyV1,
) error {
	if cfg == nil {
		return fmt.Errorf(
			"%s: config cannot be nil",
			RuntimePolicyGateVersionV1,
		)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf(
			"%s: validate canonical config: %w",
			RuntimePolicyGateVersionV1,
			err,
		)
	}

	if err := burncontrol.ValidatePolicyV1(
		policies.Burn,
	); err != nil {
		return fmt.Errorf(
			"%s: burn policy: %w",
			RuntimePolicyGateVersionV1,
			err,
		)
	}

	if err := economycontrol.ValidatePolicyV1(
		policies.Economy,
	); err != nil {
		return fmt.Errorf(
			"%s: economy policy: %w",
			RuntimePolicyGateVersionV1,
			err,
		)
	}

	if err := foundervesting.ValidatePolicyV1(
		policies.FounderVesting,
	); err != nil {
		return fmt.Errorf(
			"%s: founder vesting policy: %w",
			RuntimePolicyGateVersionV1,
			err,
		)
	}

	if err := hardforkpolicy.ValidatePolicyV1(
		policies.Hardfork,
	); err != nil {
		return fmt.Errorf(
			"%s: hardfork policy: %w",
			RuntimePolicyGateVersionV1,
			err,
		)
	}

	if err := networkfee.ValidatePolicyV1(
		policies.NetworkFee,
	); err != nil {
		return fmt.Errorf(
			"%s: network fee policy: %w",
			RuntimePolicyGateVersionV1,
			err,
		)
	}

	if err := validateCanonicalEconomyAlignmentV1(
		cfg,
		policies,
	); err != nil {
		return err
	}

	if err := validateCanonicalHardforkAlignmentV1(
		policies.Hardfork, activation,
	); err != nil {
		return err
	}

	if err := validateCanonicalFeeAccountingV1(); err != nil {
		return err
	}

	profile, err := config.ResolveNetworkProfileV1(
		cfg.ChainFile,
	)
	if err != nil {
		return fmt.Errorf(
			"%s: resolve runtime network profile: %w",
			RuntimePolicyGateVersionV1,
			err,
		)
	}

	if profile.IsExplicitMainnet {
		return fmt.Errorf(
			"%s: explicit mainnet startup blocked: "+
				"enforceable network fee denomination is not finalized",
			RuntimePolicyGateVersionV1,
		)
	}

	return nil
}

func validateCanonicalEconomyAlignmentV1(
	cfg *config.Config,
	policies RuntimePolicySetV1,
) error {
	params := consensus.LegacyParamsV1()
	if err := params.Validate(); err != nil {
		return fmt.Errorf(
			"%s: canonical consensus parameters: %w",
			RuntimePolicyGateVersionV1,
			err,
		)
	}

	if params.TotalSupplyQC !=
		blockchain.QCTotalSupplyHardCap {
		return fmt.Errorf(
			"%s: consensus/blockchain total supply mismatch: "+
				"consensus=%d blockchain=%d",
			RuntimePolicyGateVersionV1,
			params.TotalSupplyQC,
			blockchain.QCTotalSupplyHardCap,
		)
	}

	if policies.Economy.TotalSupplyQC !=
		blockchain.QCTotalSupplyHardCap ||
		policies.Burn.MaxSupplyQC !=
			blockchain.QCTotalSupplyHardCap ||
		policies.FounderVesting.MaxSupplyQC !=
			blockchain.QCTotalSupplyHardCap ||
		policies.NetworkFee.MaxSupplyQC !=
			blockchain.QCTotalSupplyHardCap {
		return fmt.Errorf(
			"%s: policy total supply declarations disagree "+
				"with canonical blockchain supply",
			RuntimePolicyGateVersionV1,
		)
	}

	if int64(cfg.TotalSupply) !=
		blockchain.QCTotalSupplyHardCap {
		return fmt.Errorf(
			"%s: config total supply mismatch: got=%d want=%d",
			RuntimePolicyGateVersionV1,
			cfg.TotalSupply,
			blockchain.QCTotalSupplyHardCap,
		)
	}

	if params.InitialSubsidyQC !=
		int64(blockchain.CanonicalInitialMiningRewardV1) ||
		policies.Economy.InitialMiningRewardQC !=
			params.InitialSubsidyQC ||
		int64(cfg.InitialReward) !=
			params.InitialSubsidyQC {
		return fmt.Errorf(
			"%s: initial mining reward declarations disagree",
			RuntimePolicyGateVersionV1,
		)
	}

	if int64(cfg.CoinbaseMaturity) !=
		policies.Economy.CoinbaseMaturityBlocks {
		return fmt.Errorf(
			"%s: coinbase maturity mismatch: config=%d policy=%d",
			RuntimePolicyGateVersionV1,
			cfg.CoinbaseMaturity,
			policies.Economy.CoinbaseMaturityBlocks,
		)
	}

	if policies.Economy.FounderTotalBasisPoints !=
		cfg.PreminePercent*100 ||
		policies.Economy.FounderLiquidBasisPoints !=
			cfg.PremineLiquidPercent*100 ||
		policies.Economy.FounderLockedBasisPoints !=
			cfg.PremineLockedPercent*100 {
		return fmt.Errorf(
			"%s: founder allocation config/policy mismatch",
			RuntimePolicyGateVersionV1,
		)
	}

	if policies.FounderVesting.FounderTotalQC !=
		blockchain.QCFounderTotalPool ||
		policies.FounderVesting.FounderLiquidQC !=
			blockchain.QCFounderLiquidPool ||
		policies.FounderVesting.FounderLockedQC !=
			blockchain.QCFounderLockedPool {
		return fmt.Errorf(
			"%s: founder allocation policy/blockchain mismatch",
			RuntimePolicyGateVersionV1,
		)
	}

	if policies.FounderVesting.CliffMonths !=
		blockchain.FounderCliffMonthsV1 ||
		policies.FounderVesting.VestingMonths !=
			blockchain.FounderVestingMonthsV1 ||
		policies.FounderVesting.MonthlyUnlockQC !=
			int64(blockchain.FounderMonthlyUnlockQCV1) ||
		policies.FounderVesting.FounderLockedQC !=
			int64(blockchain.FounderLockedAllocationV1) {
		return fmt.Errorf(
			"%s: founder vesting policy/blockchain mismatch",
			RuntimePolicyGateVersionV1,
		)
	}

	if cfg.FounderVestingCliffMonths !=
		blockchain.FounderCliffMonthsV1 ||
		cfg.FounderVestingDurationMonths !=
			blockchain.FounderVestingMonthsV1 {
		return fmt.Errorf(
			"%s: founder vesting config/blockchain mismatch",
			RuntimePolicyGateVersionV1,
		)
	}

	if strings.TrimSpace(
		policies.Burn.BurnAddressAlias,
	) != blockchain.QCBurnSinkAddress ||
		strings.TrimSpace(
			cfg.BurnAddress,
		) != blockchain.QCBurnSinkAddress {
		return fmt.Errorf(
			"%s: burn sink declaration mismatch",
			RuntimePolicyGateVersionV1,
		)
	}

	if strings.TrimSpace(cfg.MiningPoolAddress) != blockchain.QCMiningEmissionPoolAddress {
		return fmt.Errorf(
			"%s: canonical mining pool address mismatch",
			RuntimePolicyGateVersionV1,
		)
	}

	return nil
}

func validateCanonicalHardforkAlignmentV1(
	policy hardforkpolicy.HardforkPolicy,
	activation consensus.ActivationPolicyV1,
) error {

	if err := activation.Validate(); err != nil {
		return fmt.Errorf(
			"%s: canonical activation policy: %w",
			RuntimePolicyGateVersionV1,
			err,
		)
	}

	// Declarative policy never executes a fork itself. Enabled activation is
	// permitted only with the verified proof checked above.
	if policy.HardforkExecutionNow {
		return fmt.Errorf(
			"%s: control-only hardfork policy conflicts "+
				"with enabled activation",
			RuntimePolicyGateVersionV1,
		)
	}

	return nil
}

func validateCanonicalFeeAccountingV1() error {
	if strings.TrimSpace(
		blockchain.FeeConsensusVersionV1,
	) == "" {
		return fmt.Errorf(
			"%s: canonical fee accounting version unavailable",
			RuntimePolicyGateVersionV1,
		)
	}

	params := consensus.LegacyParamsV1()
	const verifiedFee int64 = 3

	result, err := consensus.ValidateBlockRewardV1(
		1,
		params.InitialSubsidyQC+verifiedFee,
		params.InitialSubsidyQC,
		verifiedFee,
		params.TotalSupplyQC,
	)
	if err != nil {
		return fmt.Errorf(
			"%s: canonical fee accounting probe: %w",
			RuntimePolicyGateVersionV1,
			err,
		)
	}

	if result.Subsidy != params.InitialSubsidyQC ||
		result.Fees != verifiedFee ||
		result.CoinbaseTotal !=
			params.InitialSubsidyQC+verifiedFee {
		return fmt.Errorf(
			"%s: canonical fee accounting probe mismatch",
			RuntimePolicyGateVersionV1,
		)
	}

	return nil
}
