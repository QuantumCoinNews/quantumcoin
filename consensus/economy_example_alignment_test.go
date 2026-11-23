package consensus_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"quantumcoin/burncontrol"
	"quantumcoin/consensus"
	"quantumcoin/economycontrol"
	"quantumcoin/foundervesting"
	"quantumcoin/networkfee"
)

// Checked-in examples are copies for operators, never a second monetary authority.
func TestCanonicalEconomyCheckedInPolicyExamples(t *testing.T) {
	cases := []struct {
		name     string
		validate func([]byte) error
	}{
		{"burn_sink_control.example.json", func(b []byte) error {
			var p burncontrol.BurnPolicy
			if err := json.Unmarshal(b, &p); err != nil {
				return err
			}
			return burncontrol.ValidatePolicyV1(p)
		}},
		{"mainnet_before_economy_control.example.json", func(b []byte) error {
			var p economycontrol.EconomyPolicy
			if err := json.Unmarshal(b, &p); err != nil {
				return err
			}
			return economycontrol.ValidatePolicyV1(p)
		}},
		{"founder_vesting_policy.example.json", func(b []byte) error {
			var p foundervesting.FounderVestingPolicy
			if err := json.Unmarshal(b, &p); err != nil {
				return err
			}
			return foundervesting.ValidatePolicyV1(p)
		}},
		{"network_fee_policy.example.json", func(b []byte) error {
			var p networkfee.NetworkFeePolicy
			if err := json.Unmarshal(b, &p); err != nil {
				return err
			}
			return networkfee.ValidatePolicyV1(p)
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "configs", c.name))
			if err != nil {
				t.Fatal(err)
			}
			if err := c.validate(data); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCanonicalEconomyPercentAndBasisPointAgreement(t *testing.T) {
	for _, v := range []struct{ percent, bp int }{
		{consensus.CanonicalFounderTotalPercentV1, consensus.CanonicalFounderTotalBasisPointsV1},
		{consensus.CanonicalFounderLiquidPercentV1, consensus.CanonicalFounderLiquidBasisPointsV1},
		{consensus.CanonicalFounderLockedPercentV1, consensus.CanonicalFounderLockedBasisPointsV1},
	} {
		if v.percent*100 != v.bp {
			t.Fatal("canonical percent/basis-point drift")
		}
	}
}
