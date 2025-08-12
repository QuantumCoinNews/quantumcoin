// api/http_server.go
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/internal"
	"quantumcoin/p2p"
	"quantumcoin/wallet"
)

// ---- Global (paket içi) durum ----
var (
	bc  *blockchain.Blockchain
	wlt *wallet.Wallet
	cfg *config.Config
)

// Init: API katmanına bağımlılıkları ver
func Init(chain *blockchain.Blockchain, defaultWallet *wallet.Wallet, conf *config.Config) {
	bc = chain
	wlt = defaultWallet
	if conf == nil {
		cfg = config.Current()
	} else {
		cfg = conf
	}
}

// ---- Başlatma (port/addr parametreli) ----

// StartHTTP: HTTP sunucusunu başlatır (addr boşsa config.HTTPPort kullanır)
// addr "8090" veya ":8090" şeklinde verilebilir.
func StartHTTP(addr string) error {
	if bc == nil {
		return fmt.Errorf("api not initialized: call api.Init first")
	}
	if addr == "" {
		addr = cfg.HTTPPort
	}
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}

	mux := http.NewServeMux()

	// Cüzdan
	mux.HandleFunc("/api/wallet/new", handleNewWallet)
	mux.HandleFunc("/api/wallet/balance/", handleBalance)

	// Transfer + Madencilik
	mux.HandleFunc("/api/send", handleSend)      // POST
	mux.HandleFunc("/api/mine", handleMineBlock) // POST

	// Zincir ve bonuslar
	mux.HandleFunc("/api/chain", handleChain)      // GET
	mux.HandleFunc("/api/ai/bonus", handleAIBonus) // GET ?address=
	mux.HandleFunc("/api/health", handleHealth)    // GET

	// Sunucu (makul timeout'lar)
	srv := &http.Server{
		Addr:              addr,
		Handler:           logMiddleware(mux),
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	fmt.Println("HTTP API started at http://localhost" + addr)
	return srv.ListenAndServe()
}

// ---- Geriye dönük uyumluluk / alias'lar ----
// Bu fonksiyonlar eski çağrıları kırmadan StartHTTP'ye delege eder.
func StartHTTPServer(port string) error { return StartHTTP(port) }
func StartAPI(port string) error        { return StartHTTP(port) }
func Start(port string) error           { return StartHTTP(port) }

// ---- Handlers ----

type sendReq struct {
	From   string  `json:"from,omitempty"` // opsiyonel; boşsa wlt kullanılır
	To     string  `json:"to"`
	Amount float64 `json:"amount"`
}

// POST /api/send
func handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if bc == nil {
		httpError(w, http.StatusServiceUnavailable, "blockchain not ready")
		return
	}

	var req sendReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, "geçersiz JSON")
		return
	}
	if req.To == "" || req.Amount <= 0 {
		httpError(w, http.StatusBadRequest, "adres ve miktar zorunludur")
		return
	}

	fromAddr := req.From
	if fromAddr == "" {
		// Varsayılan cüzdan yoksa oluştur
		if wlt == nil {
			wlt = wallet.NewWallet()
		}
		fromAddr = wlt.GetAddress()
	}

	tx, err := blockchain.NewTransaction(fromAddr, req.To, int(req.Amount), bc)
	if err != nil {
		httpError(w, http.StatusBadRequest, "işlem oluşturulamadı: "+err.Error())
		return
	}
	if err := bc.AddTransaction(tx); err != nil {
		httpError(w, http.StatusInternalServerError, "işlem havuza eklenemedi: "+err.Error())
		return
	}

	// P2P yayınını yap
	p2p.BroadcastMessage(p2p.TxMessage(tx))

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "pending",
		"from":   fromAddr,
		"to":     req.To,
		"amount": req.Amount,
		"txid":   fmt.Sprintf("%x", tx.ID),
	})
}

type mineReq struct {
	Address string `json:"address"`
	Bits    *int   `json:"bits,omitempty"` // opsiyonel: zorluk override
}

// POST /api/mine
func handleMineBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if bc == nil {
		httpError(w, http.StatusServiceUnavailable, "blockchain not ready")
		return
	}
	body, _ := io.ReadAll(r.Body)
	var req mineReq
	_ = json.Unmarshal(body, &req)

	if req.Address == "" {
		// varsayılan cüzdanla kaz
		if wlt == nil {
			wlt = wallet.NewWallet()
		}
		req.Address = wlt.GetAddress()
	}

	difficulty := cfg.DefaultDifficultyBits
	if req.Bits != nil && *req.Bits > 0 {
		difficulty = *req.Bits
	}

	block, err := bc.MineBlock(req.Address, difficulty)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// P2P yayını
	p2p.BroadcastMessage(p2p.BlockMessage(block))

	writeJSON(w, http.StatusOK, map[string]any{
		"success":    true,
		"address":    req.Address,
		"height":     bc.GetBestHeight(),
		"reward":     blockchain.GetCurrentReward(),
		"block_hash": fmt.Sprintf("%x", block.Hash),
	})
}

// GET /api/wallet/new
func handleNewWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	wlt = wallet.NewWallet()
	writeJSON(w, http.StatusOK, map[string]string{
		"address": wlt.GetAddress(),
	})
}

// GET /api/wallet/balance/{address}
func handleBalance(w http.ResponseWriter, r *http.Request) {
	if bc == nil {
		httpError(w, http.StatusServiceUnavailable, "blockchain not ready")
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/wallet/balance/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		httpError(w, http.StatusBadRequest, "address required")
		return
	}
	address := parts[0]
	balance := bc.GetBalance(address)
	writeJSON(w, http.StatusOK, map[string]any{
		"address": address,
		"balance": balance,
	})
}

// GET /api/chain
func handleChain(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"height": bc.GetBestHeight(),
		"blocks": bc.GetAllBlocks(), // gob/json için public alanlar yeterli
	})
}

// GET /api/ai/bonus?address=QC...
func handleAIBonus(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")
	bonuses := internal.ListBonuses(address)
	writeJSON(w, http.StatusOK, bonuses)
}

// GET /api/health
func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"height":   bc.GetBestHeight(),
		"httpPort": cfg.HTTPPort,
		"p2pPort":  cfg.P2PPort,
	})
}

// ---- Helpers ----

func httpError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"error":   msg,
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// Basit log middleware (isteğe bağlı)
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		fmt.Printf("%s %s %s\n", r.Method, r.URL.Path, time.Since(start))
	})
}
