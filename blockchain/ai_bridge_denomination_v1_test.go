package blockchain

import (
	"testing"
	"time"
)

func TestAITxLiteQCAtOriginHeightV1(t *testing.T) {
	bc := atomRewardTestChainV1(t, 1)
	cases := []struct {
		name   string
		height int
		units  float64
		wantQC float64
	}{
		{"legacy_one_qc", 0, 1, 1},
		{"atom_one_qc", 1, 1_000_000, 1},
		{"atom_one_atom", 1, 1, 0.000001},
		{"atom_just_below_threshold", 1, 499_999_999, 499.999999},
		{"atom_at_threshold", 1, 500_000_000, 500},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tx := &Transaction{Sender: "sender", Amount: tc.units, Timestamp: time.Unix(1, 0)}
			got := bc.ToAITxLiteAtHeightV1([]*Transaction{tx}, tc.height)
			if len(got) != 1 {
				t.Fatalf("entries=%d want=1", len(got))
			}
			if got[0].Amount != tc.wantQC {
				t.Fatalf("amount=%v want=%v QC", got[0].Amount, tc.wantQC)
			}
		})
	}
}
