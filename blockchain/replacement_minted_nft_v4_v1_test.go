package blockchain

import (
	qconsensus "quantumcoin/consensus"
	"testing"
)

func TestReplacementMintedBridgeSupportsNFTV4AtomUnitsV1(t *testing.T) {
	cases := []struct {
		name   string
		units  int64
		want   int
		reject bool
	}{
		{"normal_50_QC", 50_000_000, 50, false},
		{"final_partial_7_QC", 7_000_000, 7, false},
		{"zero", 0, 0, false},
		{"fractional_policy_subsidy", 50_000_001, 0, true},
		{"negative", -1, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ProtocolSubsidyUnitsToWholeQCV1(qconsensus.ProtocolNFTV4, tc.units)
			if tc.reject {
				if err == nil {
					t.Fatal("invalid NFT V4 subsidy accepted")
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("NFT V4 conversion: got=%d want=%d error=%v", got, tc.want, err)
			}
			atom, err := ProtocolSubsidyUnitsToWholeQCV1(qconsensus.ProtocolAtomV3, tc.units)
			if err != nil || atom != got {
				t.Fatalf("NFT V4 differs from canonical atom denomination: atom=%d nft=%d error=%v", atom, got, err)
			}
		})
	}
}
