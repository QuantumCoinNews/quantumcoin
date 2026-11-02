package blockchain

import "testing"

func TestOutputQCDebugDisplayAtBlockHeightV1(t *testing.T) {
	bc := atomRewardTestChainV1(t, 1)
	for _, tc := range []struct {
		name   string
		height int
		units  int
		want   string
	}{
		{"legacy_one_qc", 0, 1, "1"},
		{"atom_one_atom", 1, 1, "0.000001"},
		{"atom_ten_atoms", 1, 10, "0.00001"},
		{"atom_reward", 1, 50_000_000, "50"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := bc.OutputAmountQCAtHeightV1(tc.units, tc.height)
			if got != tc.want {
				t.Fatalf("display=%q want=%q QC", got, tc.want)
			}
		})
	}
}
