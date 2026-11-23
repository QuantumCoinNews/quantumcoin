package p2p

import (
	"testing"
	"time"
)

func TestNetworkViewUsesOnlyFreshRegisteredValidatedObservations(t *testing.T) {
	peersMu.Lock()
	saved := peers
	peers = map[string]*peer{"a": {}, "b": {}}
	peersMu.Unlock()
	t.Cleanup(func() { peersMu.Lock(); peers = saved; peersMu.Unlock() })
	view := NetworkViewSnapshot()
	if view.PeerCount != 2 || view.HeightKnown {
		t.Fatal("invented height")
	}
	recordValidatedPeerHeight("missing", 999)
	if NetworkViewSnapshot().HeightKnown {
		t.Fatal("unregistered peer counted")
	}
	recordValidatedPeerHeight("a", 42)
	recordValidatedPeerHeight("b", 39)
	view = NetworkViewSnapshot()
	if !view.HeightKnown || view.NetworkHeight != 42 || view.MinPeerHeight != 39 || view.ObservedPeers != 2 {
		t.Fatal(view)
	}
	peersMu.Lock()
	peers["a"].heightObserved = time.Now().Add(-11 * time.Minute)
	delete(peers, "b")
	peersMu.Unlock()
	view = NetworkViewSnapshot()
	if view.HeightKnown || view.NetworkHeight != -1 || view.PeerCount != 1 {
		t.Fatal("stale/disconnected height used", view)
	}
}
