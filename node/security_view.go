package node

import (
	"quantumcoin/p2p"
	"quantumcoin/security"
	"time"
)

func (r *Runtime) SecurityRuntimeView() map[string]map[string]any {
	chain, _ := r.Snapshot()
	view := p2p.NetworkViewSnapshot()
	age := int64(-1)
	height := -1
	if chain != nil {
		height = chain.HeightSnapshot()
		if tip := chain.TipSnapshot(); tip != nil {
			age = time.Now().Unix() - tip.Timestamp
			if age < 0 {
				age = 0
			}
		}
	}
	health := security.EvaluateNodeSyncDrift(security.NodeSyncDriftInput{LocalHeight: height, PeerCount: view.PeerCount, NetworkHeight: view.NetworkHeight, MaxPeerHeight: view.NetworkHeight, MinPeerHeight: view.MinPeerHeight, LatestBlockAgeSeconds: age, NetworkHeightUnknown: !view.HeightKnown, NetworkHeightKnown: view.HeightKnown})
	health["height"] = height
	if health["health"] == "OK" {
		health["health"] = "GOOD"
	}
	health["blockCount"] = chain.BlockCountSnapshot()
	health["networkHeightKnown"] = view.HeightKnown
	health["heightSource"] = view.Source
	if chain == nil {
		health["health"] = "UNKNOWN"
		health["syncStatus"] = "NO_CHAIN_DATA"
	}
	stopPresent := false
	if r != nil {
		r.mu.RLock()
		stopPresent = r.minerStop != nil
		r.mu.RUnlock()
	}
	sections := security.LiveEvidence.Reports(r.IsMining(), stopPresent)
	sections["nodeHealth"] = health
	return sections
}
