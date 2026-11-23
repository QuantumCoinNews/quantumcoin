package blockchain

import (
	"quantumcoin/consensus"
	"testing"
)

func TestConfirmedWholeQCRewardSeriesHasExactProtocolUnits(t *testing.T) {
	for i, rewardQC := range []int{50, 25, 12, 6, 3, 1, 1, 1} {
		height := i*int(consensus.LegacyHalvingIntervalBlocks) + 1
		for _, protocol := range []consensus.ProtocolVersion{consensus.ProtocolLegacyV1, consensus.ProtocolAtomV3, consensus.ProtocolNFTV4} {
			expected := rewardQC
			if protocol != consensus.ProtocolLegacyV1 {
				expected *= int(consensus.CanonicalAtomsPerQCV1)
			}
			got, total, err := ComposeMiningRewardUnitsV1(protocol, height, 0, int(consensus.CanonicalTotalSupplyQCV1), 1)
			if err != nil || got != expected || total != expected+1 {
				t.Fatalf("height=%d protocol=%s subsidy=%d total=%d expected=%d error=%v", height, protocol, got, total, expected, err)
			}
		}
	}
}
