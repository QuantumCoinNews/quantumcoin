package p2p

import "time"

type NetworkView struct {
	HandshakenPeers int    `json:"handshakenPeers"`
	PeerCount       int    `json:"peerCount"`
	HeightKnown     bool   `json:"networkHeightKnown"`
	NetworkHeight   int    `json:"networkHeight"`
	MinPeerHeight   int    `json:"minPeerHeight"`
	ObservedPeers   int    `json:"observedPeers"`
	Source          string `json:"heightSource"`
}

// Only fully validated peer chains feed heights; hello/claims never do.
func recordValidatedPeerHeight(addr string, height int) {
	peersMu.Lock()
	defer peersMu.Unlock()
	if p := peers[addr]; p != nil && height >= 0 {
		p.validatedHeight = height
		p.heightObserved = time.Now()
	}
}

func NetworkViewSnapshot() NetworkView {
	peersMu.Lock()
	defer peersMu.Unlock()
	view := NetworkView{PeerCount: len(peers), NetworkHeight: -1, MinPeerHeight: -1, Source: "fresh-validated-peer-chain-and-blocks"}
	handshakeMu.Lock()
	for addr := range peers {
		if handshakeOK[addr] {
			view.HandshakenPeers++
		}
	}
	handshakeMu.Unlock()
	for _, p := range peers {
		if p == nil || p.heightObserved.IsZero() || time.Since(p.heightObserved) > 10*time.Minute {
			continue
		}
		view.ObservedPeers++
		view.HeightKnown = true
		if p.validatedHeight > view.NetworkHeight {
			view.NetworkHeight = p.validatedHeight
		}
		if view.MinPeerHeight < 0 || p.validatedHeight < view.MinPeerHeight {
			view.MinPeerHeight = p.validatedHeight
		}
	}
	return view
}
