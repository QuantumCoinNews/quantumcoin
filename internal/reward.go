package internal

import "quantumcoin/blockchain"

// RewardConfig is retained for source compatibility. Monetary consensus is
// owned exclusively by blockchain.Blockchain; time-based fields are retired.
type RewardConfig struct {
	InitialReward       int
	HalvingPeriodBlocks int
	HalvingPeriodSecs   int64
	MiningPeriodSecs    int64
	GenesisUnix         int64
	TotalSupply         int
}

// CalculateRewardByHeight delegates to the canonical deterministic schedule.
// Non-canonical custom parameters fail closed instead of creating a second
// consensus reward algorithm.
func CalculateRewardByHeight(initialReward int, halvingPeriodBlocks int, height int) int {
	if initialReward != blockchain.CanonicalInitialMiningRewardV1 ||
		halvingPeriodBlocks != blockchain.CanonicalBlocksPerHalvingV1 ||
		height < 0 {
		return 0
	}

	return blockchain.RewardAtHeightV1(height)
}

// CalculateRewardByTimeNow is retired. Consensus rewards require a block
// height and must never depend on local wall-clock time.
func CalculateRewardByTimeNow(_ int, _ int64, _ int64, _ int64) int {
	return 0
}

// CalculateRewardByTimeAt is retired. The former mining-period cutoff was a
// non-deterministic alternate monetary policy and is intentionally disabled.
func CalculateRewardByTimeAt(_ int, _ int64, _ int64, _ int64, _ int64) int {
	return 0
}

// ClampToSupply delegates remaining-supply accounting to canonical consensus.
func ClampToSupply(suggested, totalMinted, totalSupply int) int {
	return blockchain.CapRewardByRemainingSupplyV1(
		suggested,
		totalMinted,
		totalSupply,
	)
}

// ComputeReward always uses the canonical height-based subsidy and supply
// cap. now and legacy time/configuration fields are deliberately ignored.
func ComputeReward(cfg RewardConfig, height int, totalMinted int, now int64) int {
	if height < 0 {
		return 0
	}

	return blockchain.ExpectedSubsidyFromStateV1(
		height,
		totalMinted,
		cfg.TotalSupply,
	)
}
