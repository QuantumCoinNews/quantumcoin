package consensus

// EmissionProjectionV1 describes issuance under the existing height-based rule.
// Seconds are a projection at target block time. They do not enter validation.
type EmissionProjectionV1 struct {
	GenesisIssuedQC    int64  `json:"genesisIssuedQc"`
	FirstHalvingHeight uint64 `json:"firstHalvingHeight"`
	LastSubsidyHeight  uint64 `json:"lastSubsidyHeight"`
	LastSubsidyQC      int64  `json:"lastSubsidyQc"`
	TargetSeconds      int64  `json:"targetSeconds"`
}

func ProjectCanonicalEmissionV1(genesisIssuedQC int64) (EmissionProjectionV1, bool) {
	if genesisIssuedQC < 0 || genesisIssuedQC > CanonicalTotalSupplyQCV1 {
		return EmissionProjectionV1{}, false
	}
	remaining := CanonicalTotalSupplyQCV1 - genesisIssuedQC
	projection := EmissionProjectionV1{
		GenesisIssuedQC: genesisIssuedQC, FirstHalvingHeight: LegacyHalvingIntervalBlocks + 1,
	}
	interval := int64(LegacyHalvingIntervalBlocks)
	for remaining > 0 {
		reward, err := LegacySubsidyAtHeight(projection.LastSubsidyHeight + 1)
		if err != nil || reward < 1 {
			return EmissionProjectionV1{}, false
		}
		blocks := (remaining-1)/reward + 1
		if blocks <= interval || reward == LegacyMinimumSubsidyQC {
			projection.LastSubsidyHeight += uint64(blocks)
			projection.LastSubsidyQC = remaining - (blocks-1)*reward
			break
		}
		remaining -= interval * reward
		projection.LastSubsidyHeight += uint64(interval)
	}
	projection.TargetSeconds = int64(projection.LastSubsidyHeight) * CanonicalBlockTimeSecondsV1
	return projection, true
}
