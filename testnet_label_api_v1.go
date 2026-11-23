package main

import (
	"net/http"

	"quantumcoin/blockchain"
	qcconfig "quantumcoin/config"
)

// QC_TESTNET_LABEL_UI_API_V1 endpoint for UI/API network labeling.
func handleNetworkLabelV1(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	chainFile := ""
	chain, runtimeConfig := nodeRuntime.Snapshot()
	if runtimeConfig != nil {
		chainFile = runtimeConfig.ChainFile
	}
	info := qcconfig.NetworkLabelForIdentityV1(chainFile, "", false)
	if id := chain.NetworkIdentitySnapshotV1(); id != nil {
		info = qcconfig.NetworkLabelForIdentityV1(chainFile, id.Network, id.GenesisDomain == blockchain.IsolatedLocalGenesisDomainV1)
		info.GenesisHash = id.GenesisHash
		if magic, ok := id.DomainMagicV2(); ok {
			info.NetMagic = magic
		}
	}

	// Identity is public; the operator's filesystem location is not.
	info.ChainFile = ""
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-QuantumCoin-Network", info.Network)
	w.Header().Set("X-QuantumCoin-Network-Label", info.Label)
	if info.IsTestnet {
		w.Header().Set("X-QuantumCoin-Testnet", "true")
	} else {
		w.Header().Set("X-QuantumCoin-Testnet", "false")
	}

	writeJSON(w, http.StatusOK, info)
}
