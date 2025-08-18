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
	"quantumcoin/wallet"
)

var (
	bc  *blockchain.Blockchain
	wlt *wallet.Wallet
	cfg *config.Config
)

// Init: API katmanına bağımlılıkları enjekte et
func Init(b *blockchain.Blockchain, w *wallet.Wallet, c *config.Config) {
	bc, wlt, cfg = b, w, c
}

// iç yardımcılar
func j(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func resolveHTTPAddr(addr string) string {
	// 1) CLI’den parametre mi?
	if addr == "" {
		// 2) ENV?
		if p := os.Getenv("HTTP_PORT"); p != "" {
			addr = p
		} else if cfg != nil && cfg.HTTPPort != "" {
			// 3) config
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
	// TODO: mempool hazır olduğunda doldurulacak
	j(w, http.StatusOK, []any{})
}

func StartHTTP(addr string) error {
	if bc == nil {
		return fmt.Errorf("api not initialized: call api.Init first")
	}
	addr = resolveHTTPAddr(addr)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)
	mux.HandleFunc("/api/blocks", listBlocks)
	mux.HandleFunc("/api/block", getBlock)
	mux.HandleFunc("/api/mempool", listMempool)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return srv.ListenAndServe()
}
