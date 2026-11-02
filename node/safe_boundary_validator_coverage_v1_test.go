package node

import (
	"testing"

	"quantumcoin/backuprestore"
	"quantumcoin/burncontrol"
	"quantumcoin/economycontrol"
	"quantumcoin/miningtransition"
	"quantumcoin/networkfee"
)

func TestQMA0059DeclaredSafeBoundaryKeysAreValidatedV1(t *testing.T) {
	t.Run("burncontrol", func(t *testing.T) {
		for _, key := range []string{
			"doesNotStartMining",
			"doesNotStopMining",
		} {
			policy := burncontrol.DefaultPolicyV1()
			policy.SafeBoundary[key] = false

			if err := burncontrol.ValidatePolicyV1(policy); err == nil {
				t.Fatalf("burn policy accepted false safe-boundary key %q", key)
			}
		}
	})

	t.Run("economycontrol", func(t *testing.T) {
		for _, key := range []string{
			"doesNotStartMining",
			"doesNotStopMining",
			"preMainnetControlOnly",
		} {
			policy := economycontrol.DefaultPolicyV1()
			policy.SafeBoundary[key] = false

			if err := economycontrol.ValidatePolicyV1(policy); err == nil {
				t.Fatalf("economy policy accepted false safe-boundary key %q", key)
			}
		}
	})

	t.Run("miningtransition", func(t *testing.T) {
		for _, key := range []string{
			"doesNotChangeMiningRewardNow",
			"doesNotStartMining",
			"doesNotStopMining",
		} {
			policy := miningtransition.DefaultPolicyV1()
			policy.SafeBoundary[key] = false

			if err := miningtransition.ValidatePolicyV1(policy); err == nil {
				t.Fatalf("mining-transition policy accepted false safe-boundary key %q", key)
			}
		}
	})

	t.Run("networkfee", func(t *testing.T) {
		policy := networkfee.DefaultPolicyV1()
		policy.SafeBoundary["doesNotChangeMiningRewardNow"] = false

		if err := networkfee.ValidatePolicyV1(policy); err == nil {
			t.Fatal("network-fee policy accepted false doesNotChangeMiningRewardNow")
		}
	})

	t.Run("backuprestore", func(t *testing.T) {
		for _, key := range []string{
			"doesNotChangeConsensus",
			"doesNotStartMining",
			"doesNotStopMining",
			"runbookOnly",
		} {
			runbook := backuprestore.DefaultRunbookV1()
			runbook.SafeBoundary[key] = false

			if err := backuprestore.ValidateRunbookV1(runbook); err == nil {
				t.Fatalf("backup/restore runbook accepted false safe-boundary key %q", key)
			}
		}
	})
}
