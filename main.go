package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/bits"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"quantumcoin/ai"
	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/game"
	"quantumcoin/internal"
	"quantumcoin/p2p"
	"quantumcoin/wallet"
	"quantumcoin/webui"
)

/* ====== ANSI renk sabitleri (sadece konsol görünümü için) ====== */

const (
	ansiGreen = "\x1b[32m"
	ansiCyan  = "\x1b[36m"
	ansiReset = "\x1b[0m"
)

/* ---------- API tipleri ---------- */

type WalletResponse struct {
	Address string `json:"address"`
}

type BalanceResponse struct {
	Balance   float64 `json:"balance"`
	Spendable float64 `json:"spendable"`
	Height    int     `json:"height"`
}

type MineRequest struct {
	Address string `json:"address"`
}

type WebMineJobResp struct {
	Challenge  string `json:"challenge"`
	Difficulty int    `json:"difficulty"`
	Miner      string `json:"miner"`
	Height     int    `json:"height"`
	Expires    int64  `json:"expires"`
}

type WebMineSubmitReq struct {
	Address   string `json:"address"`
	Challenge string `json:"challenge"`
	Nonce     uint64 `json:"nonce"`
}

type WebMineSubmitResp struct {
	Accepted bool   `json:"accepted"`
	Hash     string `json:"hash"`
	Message  string `json:"message,omitempty"`
}

// İmzalı gönderim için priv_hex destekli
type SendRequest struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Amount  int    `json:"amount"`
	PrivHex string `json:"priv_hex,omitempty"`
}

type SendResponse struct {
	Success bool   `json:"success"`
	TxID    string `json:"txid"`
	Message string `json:"message,omitempty"`
}

type BurnRequest struct {
	From   string `json:"from"`
	Amount int    `json:"amount"`
}

// /api/telemetry cevabı
type TelemetryResponse struct {
	Height        int     `json:"height"`
	BlockCount    int     `json:"block_count"`
	Peers         int     `json:"peers"`
	MinerRunning  bool    `json:"miner_running"`
	HTTPPort      string  `json:"http_port"`
	P2PPort       string  `json:"p2p_port"`
	ChainFile     string  `json:"chain_file"`
	TotalSupply   int     `json:"total_supply"`
	CurrentReward int     `json:"current_reward"`
	CPUCount      int     `json:"cpu_count"`
	GoRoutines    int     `json:"goroutines"`
	MemMB         float64 `json:"mem_mb"`
}

/* ---------- AI alert buffer ---------- */

type AIAlert struct {
	Time    time.Time `json:"time"`
	Height  int       `json:"height"`
	Address string    `json:"address"`
	Reason  string    `json:"reason"`
	Score   float64   `json:"score"`
}

type AIAlertBuffer struct {
	mu    sync.Mutex
	max   int
	items []AIAlert
}

func NewAIAlertBuffer(max int) *AIAlertBuffer {
	if max <= 0 {
		max = 100
	}
	return &AIAlertBuffer{
		max:   max,
		items: make([]AIAlert, 0, max),
	}
}

func (b *AIAlertBuffer) Add(a AIAlert) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.items) >= b.max {
		copy(b.items[0:], b.items[1:])
		b.items = b.items[:b.max-1]
	}
	b.items = append(b.items, a)
}

func (b *AIAlertBuffer) List() []AIAlert {
	b.mu.Lock()
	defer b.mu.Unlock()

	out := make([]AIAlert, len(b.items))
	copy(out, b.items)
	return out
}

const apiMaxBodyBytes int64 = 1 << 20 // 1 MB

/* ---------- Global değişkenler ---------- */

var (
	bc            *blockchain.Blockchain
	gameState     = game.NewGameState()
	cfg           *config.Config
	httpServer    *http.Server
	aiAlerts      = NewAIAlertBuffer(200)
	globalMempool *internal.Mempool
)

/* ---------- Mining döngüleri ---------- */

func startContinuousMining(miner string) {
	fmt.Printf("⛏️  Continuous mining started for %s (difficulty=%d)\n", miner, cfg.DefaultDifficultyBits)

	for {
		select {
		case <-minerStop:
			fmt.Println("🛑 Miner stopped.")
			return

		default:
			blk, err := bc.MineBlock(miner, cfg.DefaultDifficultyBits)
			if err != nil {
				log.Printf("mine error: %v", err)
				time.Sleep(500 * time.Millisecond)
				continue
			}

			// Yeni blok tüm peer'lara yayılıyor
			p2p.BroadcastMessage(p2p.BlockMessage(blk))

			// Konsol log'u (renkli)
			fmt.Printf(ansiGreen+"✅ Block #%d mined"+ansiReset+"  Hash: %s%s%s\n",
				blk.Index, ansiCyan, hex.EncodeToString(blk.Hash), ansiReset)

			// AI bonus işlemleri
			processAIBonus()

			// Zinciri diske yaz
			_ = bc.SaveToFile(cfg.ChainFile)

			// Küçük bir nefes payı
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func autosaveLoop() {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	for range t.C {
		if err := bc.SaveToFile(cfg.ChainFile); err != nil {
			log.Println("autosave error:", err)
		}
	}
}

/* web miner job state */

type webJob struct {
	Challenge  []byte
	Difficulty int
	Miner      string
	Height     int
	ExpiresAt  time.Time
}

var (
	jobMu     sync.Mutex
	curJob    *webJob
	minerStop chan struct{}
)

/* ---------- Yardımcı fonksiyonlar ---------- */

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  run [port]               - Run node (override P2P port)")
	fmt.Println("  run-mine [miner]         - Run node + API + continuous mining")
	fmt.Println("  connect [port] [addr]    - Connect to peer")
	fmt.Println("  send [from] [to] [amt]   - Send coins")
	fmt.Println("  mine [miner]             - Mine one block")
	fmt.Println("  mine-forever [miner]     - Continuous mining")
	fmt.Println("  print                    - Print chain")
	fmt.Println("  newaddr                  - Generate wallet address")
	fmt.Println("  newaddr-priv             - Generate wallet + print private key (hex)")
	fmt.Println("  api                      - Start HTTP API")
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"success": false, "message": msg})
}

func getHTTPPort() string {
	if p := os.Getenv("HTTP_PORT"); p != "" {
		return p
	}
	return cfg.HTTPPort
}

func getHTTPAddr() string {
	addr := getHTTPPort()
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}
	return addr
}

// Varsayılan adres çözümü:
// 1) %APPDATA%\QuantumCoin\wallet.json
// 2) QC_MINER / config.json
// 3) miner_address.txt
// 4) yeni üret + miner_address.txt'ye yaz
func getDefaultAddress() string {
	if app := os.Getenv("APPDATA"); strings.TrimSpace(app) != "" {
		wpath := filepath.Join(app, "QuantumCoin", "wallet.json")
		if b, err := os.ReadFile(wpath); err == nil {
			var w struct {
				Address string `json:"address"`
			}
			if json.Unmarshal(b, &w) == nil && strings.TrimSpace(w.Address) != "" {
				return strings.TrimSpace(w.Address)
			}
		}
	}
	if v := strings.TrimSpace(os.Getenv("QC_MINER")); v != "" {
		return v
	}
	if v := getMinerAddressFromConfig(); v != "" {
		return v
	}
	if data, err := os.ReadFile("miner_address.txt"); err == nil {
		if s := strings.TrimSpace(string(data)); s != "" {
			return s
		}
	}
	w := wallet.NewWallet()
	addr := w.GetAddress()
	_ = os.WriteFile("miner_address.txt", []byte(addr), 0644)
	return addr
}

func getMinerAddressFromConfig() string {
	if s := os.Getenv("QC_MINER"); strings.TrimSpace(s) != "" {
		return strings.TrimSpace(s)
	}
	b, err := os.ReadFile("config.json")
	if err != nil {
		return ""
	}
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		return ""
	}
	if v, ok := m["Miner"]; ok {
		if mm, ok := v.(map[string]any); ok {
			if addr, ok := mm["Address"].(string); ok && addr != "" {
				return addr
			}
			if addr, ok := mm["address"].(string); ok && addr != "" {
				return addr
			}
		}
	}
	if addr, ok := m["premine_address"].(string); ok && addr != "" {
		return addr
	}
	return ""
}

func ensureMinerAddress() (string, error) {
	if v := strings.TrimSpace(os.Getenv("QC_MINER")); v != "" {
		return v, nil
	}
	if v := getMinerAddressFromConfig(); v != "" {
		return v, nil
	}
	if data, err := os.ReadFile("miner_address.txt"); err == nil {
		if s := strings.TrimSpace(string(data)); s != "" {
			return s, nil
		}
	}
	w := wallet.NewWallet()
	addr := w.GetAddress()
	_ = os.WriteFile("miner_address.txt", []byte(addr), 0644)
	return addr, nil
}

/* ---------- main ---------- */

func main() {
	var err error

	// EXE klasörüne chdir
	if exe, e := os.Executable(); e == nil {
		_ = os.Chdir(filepath.Dir(exe))
	}

	// Config yükle (TEK sefer)
	cfg, err = config.Load("config.json")
	if err != nil {
		log.Fatalf("Config yüklenemedi: %v", err)
	}
	internal.SetBonusFile(cfg.BonusFile)
	// Blockchain yükle / oluştur
	if _, err = os.Stat(cfg.ChainFile); err == nil {
		bc, err = blockchain.LoadBlockchainFromFile(cfg.ChainFile)
		if err != nil {
			log.Fatalf("Blockchain yüklenemedi: %v", err)
		}
	} else {
		bc = blockchain.NewBlockchain(cfg.InitialReward, cfg.TotalSupply)
	}
	bc.SetCoinbaseMaturity(cfg.CoinbaseMaturity)

	// --- OPSİYONEL: Zincir bütünlüğü kontrolü (ileri seviye güvenlik) ---
	if strings.TrimSpace(os.Getenv("QC_STRICT_CHAIN")) == "1" {
		if err := bc.ValidateBlockchain(); err != nil {
			log.Fatalf("Chain validation failed: %v", err)
		}
	}

	// --- MEMPOOL (AI destekli global mempool) ---
	globalMempool = internal.NewMempool()

	// Auto mode: arg yoksa node + API + mining
	if len(os.Args) < 2 {
		minerAddr := getDefaultAddress()
		fmt.Printf("⛏️  Auto mode: node+api+mining -> %s (difficulty=%d)\n", minerAddr, cfg.DefaultDifficultyBits)
		minerStop = make(chan struct{})
		go startHTTPAPI()
		go autosaveLoop()
		go startContinuousMining(minerAddr)
		go trapAndShutdown()
		p := strings.TrimPrefix(cfg.P2PPort, ":")
		p2p.RunNode(p, bc)
		return
	}

	switch os.Args[1] {
	case "run":
		if len(os.Args) >= 3 {
			port := os.Args[2]
			go startHTTPAPI()
			go autosaveLoop()
			go trapAndShutdown()
			p2p.RunNode(port, bc)
		} else {
			go startHTTPAPI()
			go autosaveLoop()
			go trapAndShutdown()
			p := strings.TrimPrefix(cfg.P2PPort, ":")
			p2p.RunNode(p, bc)
		}

	case "run-mine":
		if len(os.Args) < 3 {
			fmt.Println("Usage: run-mine [miner]")
			return
		}
		miner := os.Args[2]
		minerStop = make(chan struct{})
		go startHTTPAPI()
		go autosaveLoop()
		go startContinuousMining(miner)
		go trapAndShutdown()
		p := strings.TrimPrefix(cfg.P2PPort, ":")
		p2p.RunNode(p, bc)

	case "api":
		go autosaveLoop()
		go trapAndShutdown()
		startHTTPAPI()

	case "connect":
		if len(os.Args) < 4 {
			fmt.Println("Usage: connect [port] [address]")
			return
		}
		port := os.Args[2]
		address := os.Args[3]
		go startHTTPAPI()
		go autosaveLoop()
		go trapAndShutdown()
		p2p.ConnectToPeer(port, address, bc)

	case "send":
		if len(os.Args) < 5 {
			fmt.Println("Usage: send [from] [to] [amount]")
			return
		}
		from := os.Args[2]
		to := os.Args[3]
		amount, err := strconv.Atoi(os.Args[4])
		if err != nil || amount <= 0 {
			fmt.Println("Invalid amount")
			return
		}
		tx, err := blockchain.NewTransaction(from, to, amount, bc)
		if err != nil {
			log.Println("tx build failed:", err)
			return
		}
		if err := bc.AddTransaction(tx); err != nil {
			log.Println("tx submit failed:", err)
			return
		}
		p2p.BroadcastMessage(p2p.TxMessage(tx))
		fmt.Printf("✓ Transaction accepted and broadcasted (txid=%s)\n", hex.EncodeToString(tx.ID))

	case "mine":
		if len(os.Args) < 3 {
			fmt.Println("Usage: mine [miner]")
			return
		}
		miner := os.Args[2]
		difficulty := cfg.DefaultDifficultyBits
		block, err := bc.MineBlock(miner, difficulty)
		if err != nil {
			log.Println("mining failed:", err)
			return
		}
		p2p.BroadcastMessage(p2p.BlockMessage(block))
		fmt.Printf(ansiGreen+"✅ New block mined by %s"+ansiReset+"\n", miner)
		fmt.Printf("   Hash:   %s%s%s\n", ansiCyan, hex.EncodeToString(block.Hash), ansiReset)
		fmt.Printf("   Height: %d  Reward: %d QC\n", bc.GetBestHeight(), blockchain.GetCurrentReward())
		processAIBonus()
		_ = bc.SaveToFile(cfg.ChainFile)

	case "mine-forever":
		if len(os.Args) < 3 {
			fmt.Println("Usage: mine-forever [miner]")
			return
		}
		miner := os.Args[2]
		minerStop = make(chan struct{})
		go autosaveLoop()
		go trapAndShutdown()
		startContinuousMining(miner)

	case "print":
		for _, block := range bc.Blocks {
			fmt.Printf("📦 Block #%d\n", block.Index)
			fmt.Printf("⛏️  Miner     : %s\n", block.Miner)
			fmt.Printf("🧱 Hash       : %s\n", hex.EncodeToString(block.Hash))
			fmt.Printf("🔗 PrevHash   : %s\n", hex.EncodeToString(block.PrevHash))
			fmt.Println("📝 Transactions:")
			for _, tx := range block.Transactions {
				fmt.Printf("  TxID: %s\n", hex.EncodeToString(tx.ID))
				for _, out := range tx.Outputs {
					fmt.Printf("    🔸 Amount: %d QC\n", out.Amount)
				}
			}
			fmt.Println("-------------------------------")
		}

	case "newaddr":
		w := wallet.NewWallet()
		address := w.GetAddress()
		fmt.Println("New Wallet Address:", address)

	case "newaddr-priv":
		w := wallet.NewWallet()
		address := w.GetAddress()
		fmt.Println("New Wallet Address:", address)
		fmt.Println("PrivateKey (hex):", w.ExportPrivateKeyHex())
		return

	default:
		printUsage()
	}

	if err := bc.SaveToFile(cfg.ChainFile); err != nil {
		log.Fatalf("Blockchain kaydedilemedi: %v", err)
	}
}

// --- DEV / LOCAL KONTROLLERİ ---

// QC_DEV_MODE ortam değişkeni "1", "true" veya "yes" ise dev modu açık sayıyoruz.
func isDevModeEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("QC_DEV_MODE")))
	return v == "1" || v == "true" || v == "yes"
}

// İstek sadece localhost'tan mı geldi?
func isLocalRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// Beklenmeyen formatta ise, olduğu gibi deneyelim
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

// protected: Handler'ı API token kontrolü + body limiti ile sarar.
// QC_API_TOKEN boş ise checkAPIToken her zaman true döner (dev modu).
func protected(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1) API token güvenliği
		if !checkAPIToken(w, r) {
			// checkAPIToken false döndürdüyse zaten 401/403 yazdı.
			return
		}

		// 2) Body boyut limiti (POST / PUT için)
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			r.Body = http.MaxBytesReader(w, r.Body, apiMaxBodyBytes)
		}

		// 3) Asıl handler
		h(w, r)
	}
}

/* ---------- HTTP API ---------- */

// main.go içinde, HTTP API bölümünde
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		// Tarayıcıdan gelen isteklerde QC API token’ını gönderebilmek için:
		w.Header().Set("Access-Control-Allow-Headers",
			"Content-Type, Authorization, X-QC-API-KEY")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func startHTTPAPI() {
	mux := http.NewServeMux()

	// Kısa alias
	crit := criticalLimiter

	// Temel API uçları
	mux.HandleFunc("/api/health", protected(handleHealth))
	mux.HandleFunc("/api/telemetry", protected(handleTelemetry))
	mux.HandleFunc("/api/wallet/new", protected(handleNewWallet))
	mux.HandleFunc("/api/wallet/address", protected(handleWalletAddress))
	mux.HandleFunc("/api/wallet/balance/", protected(handleBalance))
	mux.HandleFunc("/api/mine", protected(handleMineBlock))

	// *** KRİTİK: tx/send – 10 saniyede max 5 istek ***
	mux.HandleFunc("/api/tx/send",
		crit.WrapFunc("tx_send", 10*time.Second, 5, protected(handleSendTx)))

	mux.HandleFunc("/api/dev/fastmine", protected(handleFastMine))

	// AI & Game
	mux.HandleFunc("/api/ai/alerts", protected(handleAIAlerts))
	mux.HandleFunc("/api/ai/bonus", protected(handleAIBonus))
	mux.HandleFunc("/api/ai/analysis", protected(handleAIAnalysis))
	mux.HandleFunc("/api/game/score", protected(handleGameScore))
	mux.HandleFunc("/api/game/leaderboard", protected(handleLeaderboard))

	// Explorer
	mux.HandleFunc("/api/blocks", protected(handleBlocksList))
	mux.HandleFunc("/api/block", protected(handleBlockDetail))

	// Burn / stake
	mux.HandleFunc("/api/tx/burn", protected(handleBurn))
	mux.HandleFunc("/api/stake/start", protected(handleStakeStart))
	mux.HandleFunc("/api/stake/status", protected(handleStakeStatus))

	// Web miner
	mux.HandleFunc("/api/mine/job", protected(handleMineJob))

	// *** KRİTİK: mine/submit – 10 saniyede max 20 istek ***
	mux.HandleFunc("/api/mine/submit",
		crit.WrapFunc("mine_submit", 10*time.Second, 20, protected(handleMineSubmit)))

	// Miner kontrol
	// *** KRİTİK: miner/start – 10 saniyede max 3 istek ***
	mux.HandleFunc("/api/miner/start",
		crit.WrapFunc("miner_start", 10*time.Second, 3, protected(handleMinerStart)))

	// *** KRİTİK: miner/stop – 10 saniyede max 3 istek ***
	mux.HandleFunc("/api/miner/stop",
		crit.WrapFunc("miner_stop", 10*time.Second, 3, protected(handleMinerStop)))

	mux.HandleFunc("/api/miner/status", protected(handleMinerStatus))

	// Web cüzdan (SPA) – burası public kalabilir, token istemiyoruz
	if h, err := webui.Handler(); err == nil {
		mux.Handle("/", h)
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true,"service":"QuantumCoin API"}`))
		})
	}

	// --- GÜVENLİ DİNLEME ADRESİ (LOCALHOST KİLİDİ) ---

	// Eski yapını bozmamak için port yine cfg / HTTP_PORT üzerinden geliyor
	portAddr := getHTTPAddr() // örn ":8082" veya "8082"

	// Tam adresi QC_API_LISTEN ile override edebilirsin (örn: "0.0.0.0:8082")
	listenAddr := strings.TrimSpace(os.Getenv("QC_API_LISTEN"))
	if listenAddr == "" {
		// QC_API_LISTEN set edilmemişse: varsayılan 127.0.0.1:[port]
		if strings.HasPrefix(portAddr, ":") {
			listenAddr = "127.0.0.1" + portAddr
		} else {
			listenAddr = "127.0.0.1:" + portAddr
		}
	}

	// Önce CORS sarmalı, ardından global HTTPGuard (rate-limit + body size)
	var handler http.Handler = withCORS(mux)
	if defaultHTTPGuard != nil {
		handler = defaultHTTPGuard.Wrap(handler)
	}

	httpServer = &http.Server{
		Addr:              listenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	fmt.Println("HTTP API starting at http://" + listenAddr)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("http server error: %v", err)
	}
}

/* ---------- Handler yardımcıları ---------- */

// Eğer yukarıda zaten writeOK tanımlıysa, bunu SİLEBİLİRSİN (tek tanım kalsın).
func writeOK(w http.ResponseWriter, v interface{}) {
	writeJSON(w, http.StatusOK, v)
}

// ----- Basit API token kontrolü (isteğe bağlı) -----
//
// QC_API_TOKEN boş ise → güvenlik katmanı pasif (her isteğe izin).
// QC_API_TOKEN ayarlı ise → X-QC-API-KEY / Authorization / ?token / form token kontrol edilir.
func checkAPIToken(w http.ResponseWriter, r *http.Request) bool {
	expected := strings.TrimSpace(os.Getenv("QC_API_TOKEN"))
	if expected == "" {
		// Token tanımlı değil → güvenlik katmanı pasif
		return true
	}

	// 1) Özel header: X-QC-API-KEY
	token := strings.TrimSpace(r.Header.Get("X-QC-API-KEY"))

	// 2) Authorization: Bearer <token>
	if token == "" {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			token = strings.TrimSpace(auth[7:])
		}
	}

	// 3) URL query: ?token=...
	if token == "" {
		token = strings.TrimSpace(r.URL.Query().Get("token"))
	}

	// 4) Form/body fallback
	if token == "" {
		if err := r.ParseForm(); err == nil {
			if t := r.Form.Get("token"); t != "" {
				token = strings.TrimSpace(t)
			}
		}
	}

	if token == "" || token != expected {
		w.Header().Set("WWW-Authenticate", `Bearer realm="QuantumCoin API"`)
		writeError(w, http.StatusUnauthorized, "invalid or missing API token")
		return false
	}
	return true
}

/* ---------- Handler fonksiyonları ---------- */

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	// Health endpoint'i kasıtlı olarak token'sız bırakıyoruz
	writeOK(w, map[string]any{
		"ok":       true,
		"height":   bc.GetBestHeight(),
		"time":     time.Now().UTC().Format(time.RFC3339),
		"httpPort": getHTTPPort(),
	})
}

func handleNewWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	// İstersen buraya da token ekleyebiliriz ama zinciri değiştirmiyor, sadece yeni key üretiyor
	wal := wallet.NewWallet()
	writeOK(w, WalletResponse{Address: wal.GetAddress()})
}

func handleWalletAddress(w http.ResponseWriter, _ *http.Request) {
	writeOK(w, WalletResponse{Address: getDefaultAddress()})
}

func handleBalance(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 5 {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}
	address := parts[4]
	total := bc.GetBalance(address)
	spend := bc.GetSpendableBalance(address)
	writeOK(w, BalanceResponse{
		Balance:   float64(total),
		Spendable: float64(spend),
		Height:    bc.GetBestHeight(),
	})
}

func handleMineBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// --- API TOKEN GÜVENLİĞİ ---
	if !checkAPIToken(w, r) {
		return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "read body: "+err.Error())
		return
	}

	var req MineRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Address == "" {
		writeError(w, http.StatusBadRequest, "address is required")
		return
	}

	block, err := bc.MineBlock(req.Address, cfg.DefaultDifficultyBits)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	p2p.BroadcastMessage(p2p.BlockMessage(block))

	writeOK(w, map[string]any{
		"success":    true,
		"reward":     blockchain.GetCurrentReward(),
		"height":     bc.GetBestHeight(),
		"block_hash": hex.EncodeToString(block.Hash),
	})

	processAIBonus()
	_ = bc.SaveToFile(cfg.ChainFile)
}

func handleSendTx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// NOT: API token kontrolü artık protected(...) wrapper'ında.
	// Bu fonksiyonun içinde tekrar checkAPIToken çağırmıyoruz.

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "read body: "+err.Error())
		return
	}

	var req SendRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.From == "" || req.To == "" || req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "from, to, amount required")
		return
	}

	privHex := strings.TrimSpace(req.PrivHex)
	if privHex == "" {
		writeOK(w, SendResponse{
			Success: false,
			TxID:    "",
			Message: "missing priv_hex for signing",
		})
		return
	}

	tx, err := blockchain.NewTransaction(req.From, req.To, req.Amount, bc)
	if err != nil {
		writeOK(w, SendResponse{
			Success: false,
			TxID:    "",
			Message: "create tx: " + err.Error(),
		})
		return
	}

	priv, err := wallet.ImportPrivateKeyHex(privHex)
	if err != nil {
		writeOK(w, SendResponse{
			Success: false,
			TxID:    "",
			Message: "invalid priv_hex: " + err.Error(),
		})
		return
	}

	if err := tx.Sign(priv); err != nil {
		writeOK(w, SendResponse{
			Success: false,
			TxID:    "",
			Message: "sign tx: " + err.Error(),
		})
		return
	}

	if !tx.Verify() {
		writeOK(w, SendResponse{
			Success: false,
			TxID:    "",
			Message: "verify failed",
		})
		return
	}

	// --- AI destekli mempool filtresi (DoS / spam / fraud koruması) ---
	if globalMempool != nil {
		if !globalMempool.Add(tx) {
			writeOK(w, SendResponse{
				Success: false,
				TxID:    "",
				Message: "tx rejected by mempool (duplicate, capacity, or AI filter)",
			})
			return
		}
		// Şu an mempool’u sadece filtre amaçlı kullanıyoruz → hemen temizle
		_ = globalMempool.RemoveTx(tx.ID)
	}
	// --- /AI mempool ---

	// --- ÇEKİRDEK BLOCKCHAIN MEMPOOL'U ---
	if err := bc.AddTransaction(tx); err != nil {
		writeOK(w, SendResponse{
			Success: false,
			TxID:    "",
			Message: "submit tx: " + err.Error(),
		})
		return
	}

	p2p.BroadcastMessage(p2p.TxMessage(tx))

	writeOK(w, SendResponse{
		Success: true,
		TxID:    hex.EncodeToString(tx.ID),
	})
}

func handleFastMine(w http.ResponseWriter, r *http.Request) {
	// Sadece POST/GET kabul edelim (eski davranışta genelde GET’di)
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// --- DEV / LOCAL GÜVENLİK KİLİDİ ---
	// 1) QC_DEV_MODE=1 (veya true/yes) olmalı
	// 2) İstek localhost'tan gelmeli (127.0.0.1 / ::1)
	if !isDevModeEnabled() || !isLocalRequest(r) {
		writeError(w, http.StatusForbidden, "fastmine disabled (dev-only, localhost)")
		return
	}
	// --- /DEV KİLİDİ ---

	nStr := r.URL.Query().Get("n")
	addr := r.URL.Query().Get("address")
	if addr == "" {
		writeError(w, http.StatusBadRequest, "address required")
		return
	}

	n, _ := strconv.Atoi(nStr)
	if n <= 0 {
		n = 5
	}

	for i := 0; i < n; i++ {
		if _, err := bc.MineBlock(addr, cfg.DefaultDifficultyBits); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	// Son bloğu peer’lara yay
	if len(bc.Blocks) > 0 {
		last := bc.Blocks[len(bc.Blocks)-1]
		p2p.BroadcastMessage(p2p.BlockMessage(last))
	}

	_ = bc.SaveToFile(cfg.ChainFile)

	writeOK(w, map[string]any{
		"success": true,
		"mined":   n,
		"height":  bc.GetBestHeight(),
	})
}

// /api/telemetry
func handleTelemetry(w http.ResponseWriter, r *http.Request) {
	// Telemetry read-only, token şart değil – istersen:
	// if !checkAPIToken(w, r) { return }

	height := -1
	blockCount := 0
	totalSupply := 0
	currentReward := 0

	if bc != nil {
		height = bc.GetBestHeight()
		blockCount = len(bc.Blocks)
		totalSupply = bc.TotalSupply
		currentReward = blockchain.GetCurrentReward()
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	writeOK(w, TelemetryResponse{
		Height:        height,
		BlockCount:    blockCount,
		Peers:         p2p.GetPeerCount(),
		MinerRunning:  minerStop != nil,
		HTTPPort:      getHTTPPort(),
		P2PPort:       strings.TrimPrefix(cfg.P2PPort, ":"),
		ChainFile:     cfg.ChainFile,
		TotalSupply:   totalSupply,
		CurrentReward: currentReward,
		CPUCount:      runtime.NumCPU(),
		GoRoutines:    runtime.NumGoroutine(),
		MemMB:         float64(m.Alloc) / (1024 * 1024),
	})
}

/* ---------- AI & Game ---------- */

func processAIBonus() {
	if !ai.Enabled() {
		return
	}

	// son 24 saatin işlemleri
	var recentTxs []*blockchain.Transaction
	cut := time.Now().Add(-24 * time.Hour)

	for _, block := range bc.Blocks {
		for _, tx := range block.Transactions {
			if tx.Timestamp.After(cut) {
				recentTxs = append(recentTxs, tx)
			}
		}
	}

	// blockchain.Transaction -> ai.TxLite
	lite := blockchain.ToAITxLite(recentTxs)

	// anomaliler
	anoms := ai.AnalyzeTransactions(lite)

	// UI için buffer'a yaz
	for _, a := range anoms {
		aiAlerts.Add(AIAlert{
			Time:    time.Now(),
			Height:  bc.GetBestHeight(),
			Address: a.WalletAddress,
			Reason:  a.Reason,
			Score:   a.Score,
		})
	}

	// AI’den bonus önerileri al
	bonusSugs := ai.DistributeAIBonusesLite(lite)

	// sistem bonus kaydına işle
	for _, bs := range bonusSugs {
		internal.GiveBonus(bs.WalletAddress, bs.Source, bs.Amount, bs.Reason, "")
	}
}

func handleAIBonus(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")
	bonuses := internal.ListBonuses(address)
	writeOK(w, bonuses)
}

func handleAIAnalysis(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")

	var userTxs []*blockchain.Transaction
	for _, block := range bc.Blocks {
		for _, tx := range block.Transactions {
			if address == "" || tx.Sender == address {
				userTxs = append(userTxs, tx)
			}
		}
	}

	if !ai.Enabled() {
		writeOK(w, map[string]any{
			"anomaly_report":     []any{},
			"recommendations":    []any{},
			"reward_suggestions": []any{},
		})
		return
	}

	lite := blockchain.ToAITxLite(userTxs)
	anoms := ai.AnalyzeTransactions(lite)
	recs := ai.BuildWalletRecommendations(anoms)
	rewards := ai.OptimizeRewards(lite, anoms)

	writeOK(w, map[string]any{
		"anomaly_report":     anoms,
		"recommendations":    recs,
		"reward_suggestions": rewards,
	})
}

func handleAIAlerts(w http.ResponseWriter, _ *http.Request) {
	list := aiAlerts.List()
	writeOK(w, list)
}

/* ---------- Game mini endpoints ---------- */

func handleGameScore(w http.ResponseWriter, r *http.Request) {
	player := r.URL.Query().Get("player")
	score, _ := strconv.Atoi(r.URL.Query().Get("score"))
	game.HandleTelegramScore(gameState, player, score)
	writeOK(w, map[string]any{"success": true})
}

func handleLeaderboard(w http.ResponseWriter, _ *http.Request) {
	top := game.GetTopPlayers(gameState, 10)
	writeOK(w, top)
}

/* ---------- Explorer ---------- */

func handleBlocksList(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 20
	}
	total := len(bc.Blocks)
	start := total - limit
	if start < 0 {
		start = 0
	}
	type bsum struct {
		Index      int    `json:"index"`
		Hash       string `json:"hash"`
		PrevHash   string `json:"prev_hash"`
		Timestamp  int64  `json:"timestamp"`
		Miner      string `json:"miner"`
		Difficulty int    `json:"difficulty"`
		TxCount    int    `json:"tx_count"`
	}
	summaries := make([]bsum, 0, limit)
	for i := start; i < total; i++ {
		b := bc.Blocks[i]
		summaries = append(summaries, bsum{
			Index:      b.Index,
			Hash:       hex.EncodeToString(b.Hash),
			PrevHash:   hex.EncodeToString(b.PrevHash),
			Timestamp:  b.Timestamp,
			Miner:      b.Miner,
			Difficulty: b.Difficulty,
			TxCount:    len(b.Transactions),
		})
	}
	writeOK(w, summaries)
}

func handleBlockDetail(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if idxStr := q.Get("index"); idxStr != "" {
		if idx, err := strconv.Atoi(idxStr); err == nil {
			if blk := bc.GetBlockByIndex(idx); blk != nil {
				writeOK(w, blk)
				return
			}
			writeError(w, http.StatusNotFound, "block not found")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid index")
		return
	}
	if h := q.Get("hash"); h != "" {
		raw, err := hex.DecodeString(strings.TrimPrefix(h, "0x"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid hash hex")
			return
		}
		if blk := bc.GetBlockByHash(raw); blk != nil {
			writeOK(w, blk)
			return
		}
		writeError(w, http.StatusNotFound, "block not found")
		return
	}
	writeError(w, http.StatusBadRequest, "index or hash required")
}

/* ---------- Burn & stake stubs ---------- */

func handleBurn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// --- API TOKEN GÜVENLİĞİ ---
	if !checkAPIToken(w, r) {
		return
	}

	if cfg.BurnAddress == "" || cfg.BurnAddress == "QC_BURN_SINK" {
		writeError(w, http.StatusBadRequest, "burn address not configured")
		return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "read body: "+err.Error())
		return
	}

	var req BurnRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.From == "" || req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "from and positive amount required")
		return
	}

	tx, err := blockchain.NewTransaction(req.From, cfg.BurnAddress, req.Amount, bc)
	if err != nil {
		writeError(w, http.StatusBadRequest, "create tx: "+err.Error())
		return
	}

	// NOT: Şu anda burn tx'leri imzasız; gelecekte /api/tx/send benzeri priv_hex ile
	// imzalı hale getireceğiz. Şimdilik AddTransaction üzerindeki imza kontrolü
	// bu akışı zaten "güvenlik tarafında" kilitlemiş durumda.

	if err := bc.AddTransaction(tx); err != nil {
		writeError(w, http.StatusBadRequest, "submit tx: "+err.Error())
		return
	}

	p2p.BroadcastMessage(p2p.TxMessage(tx))

	writeOK(w, map[string]any{
		"success": true,
		"txid":    hex.EncodeToString(tx.ID),
	})
}

func handleStakeStart(w http.ResponseWriter, r *http.Request) {
	// Stake başlatma da hassas; token istemek mantıklı
	if !checkAPIToken(w, r) {
		return
	}

	writeOK(w, map[string]any{
		"success": false,
		"message": "stake module coming soon",
	})
}

func handleStakeStatus(w http.ResponseWriter, _ *http.Request) {
	writeOK(w, map[string]any{
		"success": false,
		"message": "stake status endpoint coming soon",
	})
}

/* ---------- Graceful shutdown ---------- */

func trapAndShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	fmt.Println("\nShutting down...")
	if minerStop != nil {
		close(minerStop)
	}
	if httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = httpServer.Shutdown(ctx)
		cancel()
	}
	if err := bc.SaveToFile(cfg.ChainFile); err != nil {
		log.Printf("save on shutdown error: %v", err)
	}
	os.Exit(0)
}

/* ---------- Web miner PoW ---------- */

func powHash(challenge []byte, nonce uint64) [32]byte {
	var nb [8]byte
	binary.LittleEndian.PutUint64(nb[:], nonce)
	return sha256.Sum256(append(challenge, nb[:]...))
}

func hasLeadingZeroBits(h []byte, need int) bool {
	if need <= 0 {
		return true
	}
	for _, b := range h {
		if need <= 0 {
			return true
		}
		if b == 0 {
			need -= 8
			continue
		}
		z := bits.LeadingZeros8(b)
		return z >= need
	}
	return need <= 0
}

func makeWebJob(addr string, diff int) *webJob {
	last := bc.GetLastBlock()
	base := append([]byte(addr), last.Hash...)
	base = append(base, byte(last.Index))
	rnd := make([]byte, 8)
	_, _ = io.ReadFull(rand.Reader, rnd)
	base = append(base, rnd...)
	ch := sha256.Sum256(base)
	return &webJob{
		Challenge:  ch[:],
		Difficulty: diff,
		Miner:      addr,
		Height:     bc.GetBestHeight(),
		ExpiresAt:  time.Now().Add(60 * time.Second),
	}
}

func handleMineJob(w http.ResponseWriter, r *http.Request) {
	addr := r.URL.Query().Get("address")
	if addr == "" {
		writeError(w, http.StatusBadRequest, "address required")
		return
	}
	diff := cfg.DefaultDifficultyBits
	j := makeWebJob(addr, diff)
	jobMu.Lock()
	curJob = j
	jobMu.Unlock()
	writeOK(w, WebMineJobResp{
		Challenge:  hex.EncodeToString(j.Challenge),
		Difficulty: j.Difficulty,
		Miner:      j.Miner,
		Height:     j.Height,
		Expires:    j.ExpiresAt.Unix(),
	})
}

func handleMineSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "read body: "+err.Error())
		return
	}
	var req WebMineSubmitReq
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Address == "" || req.Challenge == "" {
		writeError(w, http.StatusBadRequest, "address & challenge required")
		return
	}

	jobMu.Lock()
	j := curJob
	jobMu.Unlock()

	if j == nil || time.Now().After(j.ExpiresAt) {
		writeOK(w, WebMineSubmitResp{Accepted: false, Hash: "", Message: "job expired"})
		return
	}
	if req.Address != j.Miner {
		writeOK(w, WebMineSubmitResp{Accepted: false, Hash: "", Message: "address mismatch"})
		return
	}
	if !strings.EqualFold(req.Challenge, hex.EncodeToString(j.Challenge)) {
		writeOK(w, WebMineSubmitResp{Accepted: false, Hash: "", Message: "challenge mismatch"})
		return
	}

	h := powHash(j.Challenge, req.Nonce)
	if !hasLeadingZeroBits(h[:], j.Difficulty) {
		writeOK(w, WebMineSubmitResp{Accepted: false, Hash: hex.EncodeToString(h[:]), Message: "below difficulty"})
		return
	}

	go func(miner string) {
		if blk, err := bc.MineBlock(miner, cfg.DefaultDifficultyBits); err == nil {
			p2p.BroadcastMessage(p2p.BlockMessage(blk))
			processAIBonus()
			_ = bc.SaveToFile(cfg.ChainFile)
		} else {
			log.Printf("mine after accept failed: %v", err)
		}
	}(req.Address)

	writeOK(w, WebMineSubmitResp{Accepted: true, Hash: hex.EncodeToString(h[:])})
}

/* ---------- Miner control (start/stop/status) ---------- */

type minerStartReq struct {
	Address string `json:"address"`
}

func handleMinerStart(w http.ResponseWriter, r *http.Request) {
	// --- Güvenlik: Miner'ı uzaktan başlatmak için token iste ---
	if !checkAPIToken(w, r) {
		return
	}

	var addr string
	if r.Method == http.MethodPost {
		defer r.Body.Close()
		var req minerStartReq
		_ = json.NewDecoder(r.Body).Decode(&req)
		addr = strings.TrimSpace(req.Address)
	}
	if addr == "" {
		addr = strings.TrimSpace(r.URL.Query().Get("address"))
	}
	if addr == "" {
		if a, err := ensureMinerAddress(); err == nil {
			addr = a
		}
	}
	if addr == "" {
		writeError(w, http.StatusBadRequest, "address required")
		return
	}

	if minerStop != nil {
		writeOK(w, map[string]any{
			"running":    true,
			"message":    "miner already running",
			"address":    addr,
			"difficulty": cfg.DefaultDifficultyBits,
			"height":     bc.GetBestHeight(),
		})
		return
	}

	minerStop = make(chan struct{})
	go startContinuousMining(addr)

	writeOK(w, map[string]any{
		"running":    true,
		"address":    addr,
		"difficulty": cfg.DefaultDifficultyBits,
		"height":     bc.GetBestHeight(),
	})
}

func handleMinerStop(w http.ResponseWriter, r *http.Request) {
	// --- Güvenlik: Miner'ı uzaktan durdurmak için token iste ---
	if !checkAPIToken(w, r) {
		return
	}

	if minerStop == nil {
		writeOK(w, map[string]any{
			"running": false,
			"message": "miner not running",
			"height":  bc.GetBestHeight(),
		})
		return
	}
	close(minerStop)
	minerStop = nil
	writeOK(w, map[string]any{
		"running": false,
		"message": "miner stopped",
		"height":  bc.GetBestHeight(),
	})
}

func handleMinerStatus(w http.ResponseWriter, _ *http.Request) {
	writeOK(w, map[string]any{
		"running":  minerStop != nil,
		"height":   bc.GetBestHeight(),
		"bits":     cfg.DefaultDifficultyBits,
		"httpPort": getHTTPPort(),
	})
}
