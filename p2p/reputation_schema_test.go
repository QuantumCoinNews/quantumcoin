package p2p

import "testing"

func TestPeerReputationAggregateSchema(t *testing.T) {
	for _, tc := range []struct {
		score int
		want  string
	}{{100, "LOW"}, {80, "LOW"}, {79, "WATCH"}, {50, "WATCH"}, {49, "MEDIUM"}, {20, "MEDIUM"}, {19, "HIGH"}, {0, "HIGH"}} {
		ps := &PeerSecurity{reputation: map[string]int{"test": tc.score, "trusted": 100}}
		snapshot := ps.ReputationSnapshot()
		if snapshot["overallRisk"] != tc.want || snapshot["peerCount"] != 2 {
			t.Fatal(tc, snapshot)
		}
	}
	if (&PeerSecurity{}).ReputationSnapshot()["overallRisk"] != "LOW" {
		t.Fatal("empty local evidence misreported")
	}
}
