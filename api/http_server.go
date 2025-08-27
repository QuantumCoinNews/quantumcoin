package api

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/miner"  // pasif; endpoint'ler duruyor
	"quantumcoin/wallet" // address decode/utxo filtre
	"quantumcoin/webui"  // gömülü web arayüz
)

var (
	bc  *blockchain.Blockchain
	cfg *config.Config
)

// Init: API katmanına bağımlılıkları enjekte et
// İmza aynı kalsın diye ikinci parametreyi (wallet) tutuyoruz ama kullanmıyoruz.
func Init(b *blockchain.Blockchain, _ any, c *config.Config) {
	bc, cfg = b, c
}

// iç yardımcılar
func j(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func resolveHTTPAddr(addr string) string {
	if addr == "" {
		if p := os.Getenv("HTTP_PORT"); p != "" {
			addr = p
		} else if cfg != nil && cfg.HTTPPort != "" {
			addr = cfg.HTTPPort
		} else {
			addr = "8081"
		}
	}
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}
	return addr
}

// —————————————————————————————————————————————
// Basit CORS sarmalayıcı (Web Miner & Web Cüzdan)
func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// —————————————————————————————————————————————

func health(w http.ResponseWriter, _ *http.Request) {
	type resp struct {
		OK      bool   `json:"ok"`
		Version string `json:"version"`
		Time    string `json:"time"`
		Height  int    `json:"height"`
	}
	h := 0
	if bc != nil {
		h = bc.GetBestHeight()
	}
	j(w, http.StatusOK, resp{
		OK:      true,
		Version: "api.v1",
		Time:    time.Now().Format(time.RFC3339),
		Height:  h,
	})
}

// minimal explorer uçları
func listBlocks(w http.ResponseWriter, r *http.Request) {
	type blockMeta struct {
		Index    int    `json:"index"`
		Hash     string `json:"hash"`
		PrevHash string `json:"prevHash"`
		Miner    string `json:"miner"`
		TxCount  int    `json:"txCount"`
	}
	if bc == nil {
		j(w, http.StatusOK, []blockMeta{})
		return
	}
	// yeni→eski
	res := make([]blockMeta, 0, len(bc.Blocks))
	for i := len(bc.Blocks) - 1; i >= 0; i-- {
		b := bc.Blocks[i]
		res = append(res, blockMeta{
			Index:    b.Index,
			Hash:     hex.EncodeToString(b.Hash),
			PrevHash: hex.EncodeToString(b.PrevHash),
			Miner:    b.Miner,
			TxCount:  len(b.Transactions),
		})
	}
	j(w, http.StatusOK, res)
}

func getBlock(w http.ResponseWriter, r *http.Request) {
	// /api/block?id=<hashOrHeight> (şimdilik stub)
	id := r.URL.Query().Get("id")
	if id == "" {
		j(w, http.StatusBadRequest, map[string]string{"error": "missing id"})
		return
	}
	j(w, http.StatusNotFound, map[string]string{"error": "not implemented yet"})
}

func listMempool(w http.ResponseWriter, r *http.Request) {
	type txMeta struct {
		ID       string  `json:"id"`
		Sender   string  `json:"sender"`
		Amount   float64 `json:"amount"`
		InCount  int     `json:"inputs"`
		OutCount int     `json:"outputs"`
		Time     string  `json:"time"`
		Verifies bool    `json:"verifies"`
		Coinbase bool    `json:"coinbase"`
	}

	if bc == nil {
		j(w, http.StatusOK, []txMeta{})
		return
	}
	out := make([]txMeta, 0, len(bc.PendingTxs()))
	for _, tx := range bc.PendingTxs() {
		tm := tx.Timestamp.UTC().Format(time.RFC3339)
		out = append(out, txMeta{
			ID:       hex.EncodeToString(tx.ID),
			Sender:   tx.Sender,
			Amount:   tx.Amount,
			InCount:  len(tx.Inputs),
			OutCount: len(tx.Outputs),
			Time:     tm,
			Verifies: tx.Verify(),
			Coinbase: tx.IsCoinbase(),
		})
	}
	j(w, http.StatusOK, out)
}

// —————————————————————————————————————————————
// Web Miner uçları (pasif — sadece mevcut dursun)
type webMineGetResp struct {
	Challenge  string `json:"challenge"`
	Difficulty int    `json:"difficulty"`
}
type webMinePostReq struct {
	Address   string `json:"address"`
	Challenge string `json:"challenge"`
	Nonce     uint32 `json:"nonce"`
	Hash      string `json:"hash,omitempty"`
}
type webMinePostResp struct {
	Accepted  bool   `json:"accepted"`
	Hash      string `json:"hash"`
	Message   string `json:"message,omitempty"`
	Rewarded  bool   `json:"rewarded,omitempty"`
	BlockHash string `json:"blockHash,omitempty"`
}

func webMineHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ch, diff := miner.CurrentWebChallenge()
		j(w, http.StatusOK, webMineGetResp{Challenge: ch, Difficulty: diff})
		return

	case http.MethodPost:
		var req webMinePostReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			j(w, http.StatusBadRequest, map[string]string{"error": "bad json: " + err.Error()})
			return
		}
		ok, hashHex := miner.VerifyWebSolution(req.Challenge, req.Nonce, miner.WebDifficulty)
		if !ok {
			j(w, http.StatusOK, webMinePostResp{Accepted: false, Hash: hashHex, Message: "invalid or below difficulty"})
			return
		}
		// Şimdilik ödül entegrasyonu yok (pasif)
		j(w, http.StatusOK, webMinePostResp{
			Accepted: true,
			Hash:     hashHex,
			Message:  "accepted (stub)",
			Rewarded: false,
		})
		return

	case http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)
		return
	default:
		j(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// —————————————————————————————————————————————
// Web Cüzdan — API Uçları

// GET /api/address/balance?addr=...
func getAddressBalance(w http.ResponseWriter, r *http.Request) {
	addr := r.URL.Query().Get("addr")
	if addr == "" || bc == nil {
		j(w, http.StatusBadRequest, map[string]string{"error": "missing addr"})
		return
	}
	bal := bc.GetBalance(addr)
	spendable := bc.GetSpendableBalance(addr)
	j(w, http.StatusOK, map[string]any{
		"address":   addr,
		"balance":   bal,
		"spendable": spendable,
	})
}

// GET /api/address/utxos?addr=...
func getAddressUTXOs(w http.ResponseWriter, r *http.Request) {
	addr := r.URL.Query().Get("addr")
	if addr == "" || bc == nil {
		j(w, http.StatusBadRequest, map[string]string{"error": "missing addr"})
		return
	}
	pkh := wallet.Base58DecodeAddress(addr)
	type utxoItem struct {
		TxID string `json:"txid"`
		N    int    `json:"n"`
		Amt  int    `json:"amount"`
	}
	var out []utxoItem
	for txID, outs := range bc.UTXO {
		for i, o := range outs {
			if o.IsLockedWithKey(pkh) {
				out = append(out, utxoItem{TxID: txID, N: i, Amt: o.Amount})
			}
		}
	}
	j(w, http.StatusOK, out)
}

// ----- DTO Katmanı (hex-string ile konuşmak için) -----

type txInDTO struct {
	TxID      string `json:"txid"` // hex
	OutIndex  int    `json:"n"`
	Signature string `json:"signature,omitempty"` // hex(encodeSig)
	PubKey    string `json:"pubKey,omitempty"`    // hex(65B)
}

type txDTO struct {
	ID        string                         `json:"id,omitempty"` // hex
	Inputs    []txInDTO                      `json:"inputs"`
	Outputs   []blockchain.TransactionOutput `json:"outputs"`
	Timestamp string                         `json:"timestamp"` // RFC3339
	Sender    string                         `json:"sender"`
	Amount    float64                        `json:"amount"`
}

func mapTxToDTO(tx *blockchain.Transaction) txDTO {
	d := txDTO{
		ID:        hex.EncodeToString(tx.ID),
		Inputs:    make([]txInDTO, len(tx.Inputs)),
		Outputs:   tx.Outputs,
		Timestamp: tx.Timestamp.UTC().Format(time.RFC3339),
		Sender:    tx.Sender,
		Amount:    tx.Amount,
	}
	for i, in := range tx.Inputs {
		var sigHex, pubHex string
		if len(in.Signature) > 0 {
			sigHex = hex.EncodeToString(in.Signature)
		}
		if len(in.PubKey) > 0 {
			pubHex = hex.EncodeToString(in.PubKey)
		}
		d.Inputs[i] = txInDTO{
			TxID:      hex.EncodeToString(in.TxID),
			OutIndex:  in.OutIndex,
			Signature: sigHex,
			PubKey:    pubHex,
		}
	}
	return d
}

func mapDTOToTx(d txDTO) (*blockchain.Transaction, error) {
	tx := &blockchain.Transaction{
		Inputs:  make([]blockchain.TransactionInput, len(d.Inputs)),
		Outputs: d.Outputs,
		Sender:  d.Sender,
		Amount:  d.Amount,
	}
	// ID
	if d.ID != "" {
		idb, err := hex.DecodeString(d.ID)
		if err != nil {
			return nil, fmt.Errorf("bad id hex: %w", err)
		}
		tx.ID = idb
	}
	// Timestamp
	if d.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, d.Timestamp); err == nil {
			tx.Timestamp = t
		} else {
			tx.Timestamp = time.Now()
		}
	} else {
		tx.Timestamp = time.Now()
	}
	// Inputs
	for i, in := range d.Inputs {
		txidb, err := hex.DecodeString(in.TxID)
		if err != nil {
			return nil, fmt.Errorf("bad input.txid hex: %w", err)
		}
		var sigb, pubb []byte
		if in.Signature != "" {
			if b, err := hex.DecodeString(in.Signature); err == nil {
				sigb = b
			} else {
				return nil, fmt.Errorf("bad signature hex: %w", err)
			}
		}
		if in.PubKey != "" {
			if b, err := hex.DecodeString(in.PubKey); err == nil {
				pubb = b
			} else {
				return nil, fmt.Errorf("bad pubKey hex: %w", err)
			}
		}
		tx.Inputs[i] = blockchain.TransactionInput{
			TxID:      txidb,
			OutIndex:  in.OutIndex,
			Signature: sigb,
			PubKey:    pubb,
		}
	}
	return tx, nil
}

// POST /api/tx/build
// Req: { "from": "...", "to": "...", "amount": 123 }
// Res: { "tx": txDTO, "signingHashes": ["hex32", ...] }
func buildUnsignedTx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		j(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		From   string `json:"from"`
		To     string `json:"to"`
		Amount int    `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		j(w, http.StatusBadRequest, map[string]string{"error": "bad json: " + err.Error()})
		return
	}
	if bc == nil {
		j(w, http.StatusServiceUnavailable, map[string]string{"error": "blockchain not ready"})
		return
	}
	tx, err := blockchain.NewTransaction(req.From, req.To, req.Amount, bc)
	if err != nil {
		j(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	hashes := blockchain.SigningHashes(tx)

	type resp struct {
		Tx            txDTO    `json:"tx"`
		SigningHashes []string `json:"signingHashes"`
	}
	j(w, http.StatusOK, resp{Tx: mapTxToDTO(tx), SigningHashes: hashes})
}

// POST /api/tx/send — imzalı tx’i mempool’a ekler (coinbase hariç Verify şart)
func sendTx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		j(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if bc == nil {
		j(w, http.StatusServiceUnavailable, map[string]string{"error": "blockchain not ready"})
		return
	}

	var dto txDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		j(w, http.StatusBadRequest, map[string]string{"error": "bad json: " + err.Error()})
		return
	}
	tx, err := mapDTOToTx(dto)
	if err != nil {
		j(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if !tx.IsCoinbase() && !tx.Verify() {
		j(w, http.StatusBadRequest, map[string]string{"error": "invalid tx signature"})
		return
	}
	if err := bc.AddTransaction(tx); err != nil {
		j(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	j(w, http.StatusOK, map[string]any{
		"accepted": true,
		"id":       hex.EncodeToString(tx.ID),
	})
}

// GET /api/tx/status?id=<hex>
func getTxStatus(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		j(w, http.StatusBadRequest, map[string]string{"error": "missing id"})
		return
	}
	inBlock := false
	if bc != nil {
	outer:
		for _, b := range bc.Blocks {
			for _, tx := range b.Transactions {
				if hex.EncodeToString(tx.ID) == id {
					inBlock = true
					break outer
				}
			}
		}
	}
	inMempool := false
	for _, tx := range bc.PendingTxs() {
		if hex.EncodeToString(tx.ID) == id {
			inMempool = true
			break
		}
	}
	j(w, http.StatusOK, map[string]any{
		"id":        id,
		"inBlock":   inBlock,
		"inMempool": inMempool,
	})
}

// —————————————————————————————————————————————

func StartHTTP(addr string) error {
	if bc == nil {
		return fmt.Errorf("api not initialized: call api.Init first")
	}
	addr = resolveHTTPAddr(addr)

	mux := http.NewServeMux()

	// API route'lar
	mux.HandleFunc("/health", health)
	mux.HandleFunc("/api/blocks", listBlocks)
	mux.HandleFunc("/api/block", getBlock)
	mux.HandleFunc("/api/mempool", listMempool)
	mux.HandleFunc("/api/address/balance", getAddressBalance)
	mux.HandleFunc("/api/address/utxos", getAddressUTXOs)
	mux.HandleFunc("/api/tx/build", buildUnsignedTx)
	mux.HandleFunc("/api/tx/send", sendTx)
	mux.HandleFunc("/api/tx/status", getTxStatus)
	mux.HandleFunc("/api/mine", webMineHandler) // pasif

	// ⤵️ Web UI (embed) — en sonda mount et
	if h, err := webui.Handler(); err == nil {
		mux.Handle("/", h) // / ve gerisi -> web cüzdan (SPA)
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           withCORS(mux),
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return srv.ListenAndServe()
}
