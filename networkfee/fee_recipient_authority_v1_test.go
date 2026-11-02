package networkfee

import (
	"strings"
	"testing"
)

func TestFeeRecipientAuthorityV1DefaultMatchesConsensus(t *testing.T) {
	policy := DefaultPolicyV1()

	if policy.FeeRecipient != "BLOCK_PRODUCER_MINER" {
		t.Fatalf(
			"fee recipient=%q want BLOCK_PRODUCER_MINER",
			policy.FeeRecipient,
		)
	}

	if policy.MiningActiveFeeDestination !=
		"BLOCK_REWARD_PLUS_TX_FEES_TO_BLOCK_PRODUCER" {
		t.Fatalf(
			"mining-active fee destination=%q",
			policy.MiningActiveFeeDestination,
		)
	}

	if policy.AfterEmissionDoneFeeDestination !=
		"TX_FEES_ONLY_TO_BLOCK_PRODUCER" {
		t.Fatalf(
			"post-emission fee destination=%q want block producer only",
			policy.AfterEmissionDoneFeeDestination,
		)
	}

	combined := strings.ToUpper(
		policy.FeeRecipient + " " +
			policy.MiningActiveFeeDestination + " " +
			policy.AfterEmissionDoneFeeDestination,
	)

	if strings.Contains(combined, "NODE_OPERATOR") ||
		strings.Contains(combined, "NODE_OR_BLOCK_PRODUCER") {
		t.Fatalf(
			"ordinary node operator entitlement leaked into fee policy: %q",
			combined,
		)
	}
}

func TestFeeRecipientAuthorityV1RejectsOrdinaryNodeEntitlement(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*NetworkFeePolicy)
	}{
		{
			name: "fee_recipient",
			mutate: func(p *NetworkFeePolicy) {
				p.FeeRecipient =
					"BLOCK_PRODUCER_MINER_OR_NODE_OPERATOR"
			},
		},
		{
			name: "mining_active_destination",
			mutate: func(p *NetworkFeePolicy) {
				p.MiningActiveFeeDestination =
					"TX_FEES_TO_NODE_OPERATOR"
			},
		},
		{
			name: "post_emission_destination",
			mutate: func(p *NetworkFeePolicy) {
				p.AfterEmissionDoneFeeDestination =
					"TX_FEES_ONLY_TO_NODE_OR_BLOCK_PRODUCER"
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			policy := DefaultPolicyV1()
			tc.mutate(&policy)

			if err := ValidatePolicyV1(policy); err == nil {
				t.Fatal(
					"ordinary node operator fee entitlement must be rejected",
				)
			}
		})
	}
}
