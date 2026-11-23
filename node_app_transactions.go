package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"quantumcoin/blockchain"
	"quantumcoin/p2p"
	"strings"
)

// QC_TX_STATUS_API_V1
// GET /api/tx/status?id=<txid>
// Read-only tx lookup used by Telegram Game payout precheck.
func handleTxStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	reloadAPIChainFromDiskIfChanged()

	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		id = strings.TrimSpace(r.URL.Query().Get("txid"))
	}
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}

	inBlock := false
	inMempool := false
	confirmedHeight := -1

	bc, _ := nodeRuntime.Snapshot()
	if bc != nil {
		location :=
			bc.TransactionLocationSnapshot(
				id,
			)

		inBlock = location.InBlock
		inMempool = location.InMempool
		confirmedHeight = location.ConfirmedHeight
	}

	status := "unknown"
	if inBlock {
		status = "confirmed"
	} else if inMempool {
		status = "pending"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"id":              id,
		"txid":            id,
		"status":          status,
		"inBlock":         inBlock,
		"inMempool":       inMempool,
		"confirmedHeight": confirmedHeight,
	})
}

func handleSendTx(w http.ResponseWriter, r *http.Request) {
	serveSignedTransactionV1(w, r, false)
}

func handleBurn(w http.ResponseWriter, r *http.Request) {
	serveSignedTransactionV1(w, r, true)
}

// HTTP never receives wallet keys and never signs. The wallet constructs and
// signs locally against its trusted chain view before submitting this envelope.
func serveSignedTransactionV1(w http.ResponseWriter, r *http.Request, burn bool) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "method not allowed")
		return
	}
	defer r.Body.Close()
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, apiMaxBodyBytes))
	if err != nil {
		if !writeRequestBodyTooLargeV1(w, err) {
			writeError(w, 400, "invalid signed transaction body")
		}
		return
	}
	var req SendRequest
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, 400, "only a signed transaction envelope is accepted; sign in your wallet")
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeError(w, 400, "invalid signed transaction envelope")
		return
	}
	tx := req.Transaction
	if !boundedSignedEnvelopeV1(tx) {
		writeError(w, 400, "valid owner-signed transaction required")
		return
	}
	isBurn := false
	for _, out := range tx.Outputs {
		if blockchain.IsCanonicalBurnOutputV1(out) {
			isBurn = true
		}
	}
	if isBurn != burn {
		writeError(w, 400, "transaction does not match send/burn intent")
		return
	}
	bc, _ := nodeRuntime.Snapshot()
	if bc == nil {
		writeError(w, 503, "node runtime unavailable")
		return
	}
	if !bc.VerifyTransaction(tx) {
		writeError(w, 400, "valid owner-signed transaction for this chain required")
		return
	}
	if err := bc.AddTransaction(tx); err != nil {
		writeError(w, 400, "signed transaction rejected by canonical validation")
		return
	}
	nodeRuntime.BroadcastP2P(p2p.TxMessage(tx))
	writeOK(w, SendResponse{Success: true, TxID: hex.EncodeToString(tx.ID)})
}

func boundedSignedEnvelopeV1(tx *blockchain.Transaction) bool {
	if tx == nil || tx.IsCoinbase() || len(tx.ID) != 32 || len(tx.Sender) > 128 || len(tx.Inputs) > 128 || len(tx.Outputs) > 128 {
		return false
	}
	for _, input := range tx.Inputs {
		if len(input.TxID) != 32 || len(input.Signature) == 0 || len(input.Signature) > 80 || len(input.PubKey) != 65 {
			return false
		}
	}
	for _, output := range tx.Outputs {
		if len(output.PubKeyHash) > 64 {
			return false
		}
	}
	return true
}

func handleStakeStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeError(w, 405, "method not allowed")
		return
	}
	serveStakeUnavailableV1(w)
}
func handleStakeStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, 405, "method not allowed")
		return
	}
	serveStakeUnavailableV1(w)
}
func serveStakeUnavailableV1(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotImplemented, map[string]any{"success": false, "available": false, "feature": "staking", "code": "FEATURE_UNAVAILABLE", "message": "staking is not implemented; no stake was created"})
}

func handleCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, 405, "method not allowed")
		return
	}
	writeOK(w, map[string]any{"version": "QC_API_CAPABILITIES_V1", "signing": "wallet-local", "signed_transactions": true, "http_wallet_creation": false, "staking": false, "score_submission": "retired", "game_backend": "telegram_game", "game_leaderboard_path": "/api/leaderboard/top"})
}
