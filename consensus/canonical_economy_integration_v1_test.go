package consensus_test

import (
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/burncontrol"
	"quantumcoin/config"
	"quantumcoin/consensus"
	"quantumcoin/economycontrol"
	"quantumcoin/foundervesting"
	"quantumcoin/miningtransition"
	"quantumcoin/networkfee"
)

func TestCanonicalEconomyCompatibilityAliasesV1(t *testing.T) {
	if blockchain.QCTotalSupplyHardCap != consensus.CanonicalTotalSupplyQCV1 {
		t.Fatal("blockchain total supply alias diverged")
	}

	if int64(blockchain.CanonicalInitialMiningRewardV1) !=
		consensus.CanonicalInitialMiningRewardQCV1 {
		t.Fatal("blockchain initial reward alias diverged")
	}

	if economycontrol.TotalSupplyQCV1 !=
		consensus.CanonicalTotalSupplyQCV1 {
		t.Fatal("economycontrol total supply alias diverged")
	}

	if economycontrol.InitialMiningRewardQCV1 !=
		consensus.CanonicalInitialMiningRewardQCV1 {
		t.Fatal("economycontrol initial reward alias diverged")
	}

	if economycontrol.CoinbaseMaturityBlocksV1 !=
		int64(consensus.CanonicalCoinbaseMaturityBlocksV1) {
		t.Fatal("economycontrol maturity alias diverged")
	}

	if foundervesting.MaxSupplyQCV1 !=
		consensus.CanonicalTotalSupplyQCV1 ||
		foundervesting.FounderTotalQCV1 !=
			consensus.CanonicalFounderTotalQCV1 ||
		foundervesting.FounderLiquidQCV1 !=
			consensus.CanonicalFounderLiquidQCV1 ||
		foundervesting.FounderLockedQCV1 !=
			consensus.CanonicalFounderLockedQCV1 ||
		foundervesting.MonthlyUnlockQCV1 !=
			consensus.CanonicalFounderMonthlyUnlockQCV1 {
		t.Fatal("foundervesting canonical aliases diverged")
	}

	if burncontrol.MaxSupplyQCV1 !=
		consensus.CanonicalTotalSupplyQCV1 ||
		burncontrol.DefaultBurnAddressV1 !=
			consensus.CanonicalBurnSinkAddressV1 {
		t.Fatal("burncontrol canonical aliases diverged")
	}

	if miningtransition.MaxSupplyQCV1 !=
		consensus.CanonicalTotalSupplyQCV1 ||
		miningtransition.InitialRewardQCV1 !=
			consensus.CanonicalInitialMiningRewardQCV1 {
		t.Fatal("miningtransition canonical aliases diverged")
	}

	if networkfee.MaxSupplyQCV1 !=
		consensus.CanonicalTotalSupplyQCV1 {
		t.Fatal("networkfee total supply alias diverged")
	}

	cfg := config.Default()

	if int64(cfg.TotalSupply) !=
		consensus.CanonicalTotalSupplyQCV1 ||
		int64(cfg.InitialReward) !=
			consensus.CanonicalInitialMiningRewardQCV1 ||
		cfg.CoinbaseMaturity !=
			consensus.CanonicalCoinbaseMaturityBlocksV1 {
		t.Fatal("config canonical economy defaults diverged")
	}

	if cfg.PreminePercent !=
		consensus.CanonicalFounderTotalPercentV1 ||
		cfg.PremineLiquidPercent !=
			consensus.CanonicalFounderLiquidPercentV1 ||
		cfg.PremineLockedPercent !=
			consensus.CanonicalFounderLockedPercentV1 ||
		cfg.FounderVestingCliffMonths !=
			consensus.CanonicalFounderCliffMonthsV1 ||
		cfg.FounderVestingDurationMonths !=
			consensus.CanonicalFounderVestingMonthsV1 {
		t.Fatal("config founder canonical defaults diverged")
	}

	if cfg.MiningPoolAddress !=
		consensus.CanonicalMiningEmissionPoolAddressV1 ||
		cfg.BurnAddress !=
			consensus.CanonicalBurnSinkAddressV1 {
		t.Fatal("config canonical economy addresses diverged")
	}
}
