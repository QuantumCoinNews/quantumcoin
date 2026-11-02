package blockchain

import "time"

// NFT_SAFE_BOUNDARY_V1
//
// Random/probabilistic NFT drops from block mining are disabled.
// Final NFT rewards must come from Telegram Game monthly candidate pool,
// AI risk review, and admin/admin-policy approval.
func nftDropEligible(
	height int64,
	blockHash []byte,
) (bool, string) {
	return false,
		"NFT_SAFE_BOUNDARY_V1_DISABLED_MONTHLY_POLICY_REQUIRED"
}

// ComputeRewardBridge builds the compatibility reward breakdown for a
// specific candidate block height.
//
// The base subsidy is derived from the canonical height-aware consensus
// calculation and is capped by the remaining supply. Transaction fees are
// conserved value and are added to the miner payment without increasing
// newly minted supply.
func ComputeRewardBridge(
	height int64,
	now int64,
	totalMintedQC int64,
	blockHash []byte,
	fees int64,
	_ any,
) RewardBreakdown {
	if now == 0 {
		now = time.Now().Unix()
	}

	effectiveHeight := height
	if effectiveHeight < 1 {
		effectiveHeight = 1
	}

	effectiveMinted := totalMintedQC
	if effectiveMinted < 0 {
		effectiveMinted = 0
	}

	base := int64(
		ExpectedSubsidyFromStateV1(
			int(effectiveHeight),
			int(effectiveMinted),
			int(QCTotalSupplyHardCap),
		),
	)

	effectiveFees := fees
	if effectiveFees < 0 {
		effectiveFees = 0
	}

	toMiner := base + effectiveFees
	if toMiner < base {
		// Defensive overflow guard. Invalid fee input must not create
		// an artificial negative or wrapped miner payment.
		toMiner = base
		effectiveFees = 0
	}

	rb := RewardBreakdown{
		Height:        effectiveHeight,
		Timestamp:     now,
		BaseSubsidy:   base,
		AnnualBonus:   0,
		FeesCollected: effectiveFees,
		ToMiner:       toMiner,
		ToStakers:     0,
		ToDev:         0,
		ToBurn:        0,
		ToCommunity:   0,
		Total:         toMiner,
	}

	ok, detail := nftDropEligible(
		effectiveHeight,
		blockHash,
	)

	rb.NFTAwarded = ok
	rb.NFTDeterminism = detail

	return rb
}

func (rb RewardBreakdown) MetadataKV() map[string]string {
	return RewardMetadataKV(rb)
}
