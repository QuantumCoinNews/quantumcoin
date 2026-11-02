package main

import (
	"net/http"

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
	if cfg != nil {
		chainFile = cfg.ChainFile
	}

	info := qcconfig.GetNetworkLabelInfoV1(chainFile)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-QuantumCoin-Network", info.Network)
	w.Header().Set("X-QuantumCoin-Network-Label", info.Label)
	if info.IsTestnet {
		w.Header().Set("X-QuantumCoin-Testnet", "true")
	} else {
		w.Header().Set("X-QuantumCoin-Testnet", "false")
	}

	writeJSON(w, http.StatusOK, info)
}
