package p2p

import (
	"testing"
	"time"
)

func TestPeerReputationEngineV1AdjustsBanAndSnapshot(t *testing.T) {
	ps := NewPeerSecurity()
	ip := "192.0.2.16"

	if got := ps.Reputation(ip); got != 100 {
		t.Fatalf("initial reputation mismatch: got=%d want=100", got)
	}

	if got := ps.AdjustReputation(ip, -85); got != 15 {
		t.Fatalf("adjusted reputation mismatch: got=%d want=15", got)
	}

	status := ps.ReputationStatus(ip)
	if status == "" {
		t.Fatalf("reputation status must not be empty")
	}

	ps.Ban(ip, time.Minute)
	if !ps.IsBanned(ip) {
		t.Fatalf("peer should be banned after Ban")
	}

	if err := ps.OnConnect(ip); err != ErrBannedPeer {
		t.Fatalf("banned peer OnConnect error mismatch: got=%v want=%v", err, ErrBannedPeer)
	}

	snap := ps.ReputationSnapshot()

	if snap["version"] != "PEER_REPUTATION_ENGINE_V1" {
		t.Fatalf("snapshot version mismatch: got=%v", snap["version"])
	}
	if snap["peers"] == nil {
		t.Fatalf("snapshot peers must exist")
	}

	safe, ok := snap["safeBoundary"].(map[string]bool)
	if !ok {
		t.Fatalf("safeBoundary type mismatch: %T", snap["safeBoundary"])
	}

	for _, key := range []string{
		"changesChain",
		"changesBalances",
		"changesConsensus",
		"changesTotalSupply",
		"performsTransfer",
		"startsMining",
		"replacesChain",
	} {
		if safe[key] {
			t.Fatalf("safeBoundary.%s must be false", key)
		}
	}
}

func TestPeerReputationEngineV1BoundsScore(t *testing.T) {
	ps := NewPeerSecurity()
	ip := "192.0.2.17"

	if got := ps.AdjustReputation(ip, 999); got != 100 {
		t.Fatalf("upper bound mismatch: got=%d want=100", got)
	}

	if got := ps.AdjustReputation(ip, -999); got != 0 {
		t.Fatalf("lower bound mismatch: got=%d want=0", got)
	}
}
