package internal

import (
	"os"
	"strings"
	"testing"

	"quantumcoin/blockchain"
)

func TestInternalRewardHeightMatchesCanonicalV1(t *testing.T) {
	for _, height := range []int{0, 1, 2, blockchain.CanonicalBlocksPerHalvingV1, blockchain.CanonicalBlocksPerHalvingV1 + 1, 10 * blockchain.CanonicalBlocksPerHalvingV1} {
		got := CalculateRewardByHeight(
			blockchain.CanonicalInitialMiningRewardV1,
			blockchain.CanonicalBlocksPerHalvingV1,
			height,
		)
		want := blockchain.RewardAtHeightV1(height)
		if got != want {
			t.Fatalf("height=%d got=%d want=%d", height, got, want)
		}
	}
}

func TestInternalRewardRejectsNonCanonicalParametersV1(t *testing.T) {
	if got := CalculateRewardByHeight(999, 1, 1); got != 0 {
		t.Fatalf("custom reward parameters got=%d want=0", got)
	}
	if got := CalculateRewardByTimeNow(50, 1, 1, 1); got != 0 {
		t.Fatalf("retired time-now reward got=%d want=0", got)
	}
	if got := CalculateRewardByTimeAt(50, 1, 1, 1, 2); got != 0 {
		t.Fatalf("retired time-at reward got=%d want=0", got)
	}
}

func TestInternalComputeRewardMatchesCanonicalStateV1(t *testing.T) {
	cases := []struct{ height, minted, supply int }{
		{1, 0, 0},
		{blockchain.CanonicalBlocksPerHalvingV1 + 1, 0, 0},
		{1, 25_499_993, 25_500_000},
		{1, 25_500_000, 25_500_000},
	}
	for _, tc := range cases {
		cfg := RewardConfig{InitialReward: 999, HalvingPeriodBlocks: 1, HalvingPeriodSecs: 1, MiningPeriodSecs: 1, GenesisUnix: 1, TotalSupply: tc.supply}
		got := ComputeReward(cfg, tc.height, tc.minted, 1)
		want := blockchain.ExpectedSubsidyFromStateV1(tc.height, tc.minted, tc.supply)
		if got != want {
			t.Fatalf("state=%+v got=%d want=%d", tc, got, want)
		}
	}
}

func TestInternalClampDelegatesCanonicalSupplyCapV1(t *testing.T) {
	for _, tc := range [][3]int{{50, 0, 25_500_000}, {50, 25_499_993, 25_500_000}, {50, 25_500_000, 25_500_000}, {-1, 0, 25_500_000}} {
		got := ClampToSupply(tc[0], tc[1], tc[2])
		want := blockchain.CapRewardByRemainingSupplyV1(tc[0], tc[1], tc[2])
		if got != want {
			t.Fatalf("case=%v got=%d want=%d", tc, got, want)
		}
	}
}

func TestInternalRewardSourceHasSingleCanonicalAuthorityV1(t *testing.T) {
	source, err := os.ReadFile("reward.go")
	if err != nil {
		t.Fatalf("read reward.go: %v", err)
	}
	text := string(source)
	for _, marker := range []string{"quantumcoin/blockchain", "blockchain.RewardAtHeightV1", "blockchain.ExpectedSubsidyFromStateV1", "blockchain.CapRewardByRemainingSupplyV1"} {
		if !strings.Contains(text, marker) {
			t.Fatalf("canonical reward marker missing: %s", marker)
		}
	}
	for _, marker := range []string{"math.Pow", "time.Now", "elapsed :=", "miningPeriodSecs", "HalvingPeriodSecs > 0"} {
		if strings.Contains(text, marker) {
			t.Fatalf("legacy time-based reward marker remains: %s", marker)
		}
	}
}
