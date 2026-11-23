package blockchain

import (
	"quantumcoin/consensus"
	"testing"
)

func TestPostEmissionRewardUnitsNeverRemintAcrossProtocols(t *testing.T) {
	for _, protocol := range []consensus.ProtocolVersion{consensus.ProtocolLegacyV1, consensus.ProtocolAtomV3, consensus.ProtocolNFTV4} {
		for _, fees := range []int{0, 1, 3} {
			subsidy, total, err := ComposeMiningRewardUnitsV1(protocol, 1, int(QCTotalSupplyHardCap), int(QCTotalSupplyHardCap), fees)
			if err != nil || subsidy != 0 || total != fees {
				t.Fatalf("protocol=%s fees=%d subsidy=%d total=%d err=%v", protocol, fees, subsidy, total, err)
			}
		}
	}
}
