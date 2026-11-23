package main

import (
	"net/http"
	"quantumcoin/blockchain"
	"quantumcoin/consensus"
	"quantumcoin/wallet"
	"strconv"
	"strings"
)

// Key generation belongs to the local wallet, which durably saves its key.
func handleNewWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeError(w, 405, "method not allowed")
		return
	}
	writeJSON(w, http.StatusGone, map[string]any{"success": false, "code": "WALLET_CREATION_LOCAL_ONLY", "message": "Create and back up a wallet locally with newaddr or the native wallet; HTTP never creates wallet keys."})
}

func handleWalletAddress(w http.ResponseWriter, r *http.Request) {
	if r == nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	selection, err := resolveDefaultMinerAddressV1()
	if err != nil {
		writeError(
			w,
			http.StatusServiceUnavailable,
			err.Error(),
		)
		return
	}

	writeOK(w, WalletResponse{Address: selection.Address})
}

func handleBalance(w http.ResponseWriter, r *http.Request) {
	if r == nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	reloadAPIChainFromDiskIfChanged()
	bc, _ := nodeRuntime.Snapshot()
	handleBalanceForChain(w, r, bc)
}

func handleBalanceForChain(
	w http.ResponseWriter,
	r *http.Request,
	chain *blockchain.Blockchain,
) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 5 {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	address := parts[4]
	if address == "" || !wallet.ValidateAddress(address) {
		writeError(w, http.StatusBadRequest, "invalid address")
		return
	}

	if chain == nil {
		writeError(
			w,
			http.StatusServiceUnavailable,
			"blockchain unavailable",
		)
		return
	}

	totalAtoms, spendableAtoms, height, err := chain.BalanceAtomsSnapshot(address)
	if err != nil {
		writeError(w, 503, "exact balance unavailable")
		return
	}
	totalQC, spendableQC := float64(totalAtoms)/float64(consensus.CanonicalAtomsPerQCV1), float64(spendableAtoms)/float64(consensus.CanonicalAtomsPerQCV1)

	writeOK(w, BalanceResponse{
		Balance:      totalQC,
		BalanceAtoms: strconv.FormatInt(totalAtoms, 10), SpendableAtoms: strconv.FormatInt(spendableAtoms, 10),
		BalanceQc: consensus.FormatAtomsAsQCV1(totalAtoms), SpendableQc: consensus.FormatAtomsAsQCV1(spendableAtoms), AtomsPerQc: "1000000",
		Spendable: spendableQC,
		Height:    height,
	})
}
