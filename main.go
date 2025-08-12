// main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"quantumcoin/ai"
	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/game"
	"quantumcoin/internal"
	"quantumcoin/p2p"
	"quantumcoin/wallet"
)

// --- API Veri Tipleri ---
type WalletResponse struct {
	Address string `json:"address"`
}
type BalanceResponse struct {
	Balance   float64 `json:"balance"`   // toplam UTXO
	Spendable float64 `json:"spendable"` // olgunlaşma dâhil harcanabilir
	Height    int     `json:"height"`    // zincir yüksekliği
}
type MineRequest struct {
	Address string `json:"address"`
}

// --- (YENİ) Transfer endpoint türleri ---
type SendRequest struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Amount int    `json:"amount"`
}
type SendResponse struct {
	Success bool   `json:"success"`
	TxID    string `json:"txid"`
	Message string `json:"message,omitempty"`
}

// --- GLOBAL ---
var (
	bc         *blockchain.Blockchain
	gameState  = game.NewGameState()
	cfg        *config.Config
	httpServer *http.Server
)

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  run [port]               - Run node on port (override config P2P port)")
	fmt.Println("  connect [port] [address] - Connect to peer")
	fmt.Println("  send [from] [to] [amt]   - Send coins")
	fmt.Println("  mine [miner]             - Mine a new block")
	fmt.Println("  print                    - Print blockchain")
	fmt.Println("  newaddr                  - Generate a new wallet address")
	fmt.Println("  api                      - Start HTTP API (config/ENV HTTP port)")
}

// ---- JSON yardımcıları ----
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"success": false, "message": msg})
}
func writeOK(w http.ResponseWriter, v interface{}) { writeJSON(w, http.StatusOK, v) }

// ---- HTTP Port çözümleme (ENV > config) ----
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

func main() {
	var err error

	// 1) Config yükle
	cfg, err = config.Load("config.json")
	if err != nil {
		log.Fatalf("Config yüklenemedi: %v", err)
	}

	// (YENİ) Bonus dosya yolunu config ile hizala (internal paketi)
	internal.SetBonusFile(cfg.BonusFile)

	// 2) Zinciri yükle ya da oluştur
	if _, err = os.Stat(cfg.ChainFile); err == nil {
		bc, err = blockchain.LoadBlockchainFromFile(cfg.ChainFile)
		if err != nil {
			log.Fatalf("Blockchain yüklenemedi: %v", err)
		}
	} else {
		bc = blockchain.NewBlockchain(cfg.InitialReward, cfg.TotalSupply)
	}

	// (YENİ) config’teki coinbase olgunlaşmasını uygula
	bc.SetCoinbaseMaturity(cfg.CoinbaseMaturity)

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "run":
		// CLI port varsa config’i override et
		if len(os.Args) >= 3 {
			port := os.Args[2]
			go startHTTPAPI() // HTTP port: ENV > config
			go autosaveLoop()
			// Graceful shutdown sinyali
			go trapAndShutdown()
			p2p.RunNode(port, bc) // ":" + port içeride ekleniyor
		} else {
			// Config’teki P2P portu kullan
			go startHTTPAPI()
			go autosaveLoop()
			go trapAndShutdown()
			p := strings.TrimPrefix(cfg.P2PPort, ":")
			p2p.RunNode(p, bc)
		}

	case "api":
		go autosaveLoop()
		// API tek başına (CTRL+C ile kapanır)
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
			log.Println("Transaction creation failed:", err)
			return
		}
		if err := bc.AddTransaction(tx); err != nil {
			log.Println("Transaction failed:", err)
			return
		}
		// P2P yayını
		p2p.BroadcastMessage(p2p.TxMessage(tx))
		fmt.Printf("✓ Transaction accepted and broadcasted (txid=%x)\n", tx.ID)

	case "mine":
		if len(os.Args) < 3 {
			fmt.Println("Usage: mine [miner]")
			return
		}
		miner := os.Args[2]
		difficulty := cfg.DefaultDifficultyBits
		block, err := bc.MineBlock(miner, difficulty)
		if err != nil {
			log.Println("Mining failed:", err)
			return
		}
		// P2P yayını
		p2p.BroadcastMessage(p2p.BlockMessage(block))

		fmt.Printf("✅ New block mined by %s\n", miner)
		fmt.Printf("   Hash:   %x\n", block.Hash)
		fmt.Printf("   Height: %d\n", bc.GetBestHeight())
		fmt.Printf("   Reward: %d QC\n", blockchain.GetCurrentReward())
		processAIBonus()
		_ = bc.SaveToFile(cfg.ChainFile)

	case "print":
		for _, block := range bc.Blocks {
			fmt.Printf("📦 Block #%d\n", block.Index)
			fmt.Printf("⛏️  Miner     : %s\n", block.Miner)
			fmt.Printf("🧱 Hash       : %x\n", block.Hash)
			fmt.Printf("🔗 PrevHash   : %x\n", block.PrevHash)
			fmt.Println("📝 Transactions:")
			for _, tx := range block.Transactions {
				fmt.Printf("  TxID: %x\n", tx.ID)
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

	default:
		printUsage()
	}

	// Tek seferlik komutlar için çıkmadan kaydet
	if err := bc.SaveToFile(cfg.ChainFile); err != nil {
		log.Fatalf("Blockchain kaydedilemedi: %v", err)
	}
}

// --- periyodik autosave (daemon modunda iş görür) ---
func autosaveLoop() {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for range t.C {
		if err := bc.SaveToFile(cfg.ChainFile); err != nil {
			log.Println("autosave error:", err)
		}
	}
}

// --- CORS middleware (geliştirme için basit) ---
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Basit, tüm originlere izin (dev)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- HTTP API ---
func startHTTPAPI() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/wallet/new", handleNewWallet)
	mux.HandleFunc("/api/wallet/balance/", handleBalance)
	mux.HandleFunc("/api/mine", handleMineBlock)
	// (ZATEN) REST transfer uç noktası
	mux.HandleFunc("/api/tx/send", handleSendTx)
	// (YENİ) hızlı madencilik (test/dev kolaylığı için)
	mux.HandleFunc("/api/dev/fastmine", handleFastMine)

	mux.HandleFunc("/api/ai/bonus", handleAIBonus)
	mux.HandleFunc("/api/ai/analysis", handleAIAnalysis)
	mux.HandleFunc("/api/game/score", handleGameScore)
	mux.HandleFunc("/api/game/leaderboard", handleLeaderboard)

	addr := getHTTPAddr()
	httpServer = &http.Server{
		Addr:    addr,
		Handler: withCORS(mux),
	}

	fmt.Println("HTTP API starting at http://localhost" + addr)
	// ListenAndServe bloklar; run/api komutunda zaten goroutine ile çağrılıyor
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("http server error: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeOK(w, map[string]any{
		"ok":       true,
		"height":   bc.GetBestHeight(),
		"time":     time.Now().UTC().Format(time.RFC3339),
		"httpPort": getHTTPPort(),
	})
}

func handleNewWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" && r.Method != "GET" {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	wal := wallet.NewWallet()
	res := WalletResponse{Address: wal.GetAddress()}
	writeOK(w, res)
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
	res := BalanceResponse{
		Balance:   float64(total),
		Spendable: float64(spend),
		Height:    bc.GetBestHeight(),
	}
	writeOK(w, res)
}

func handleMineBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, _ := io.ReadAll(r.Body)
	var req MineRequest
	_ = json.Unmarshal(body, &req)
	if req.Address == "" {
		writeError(w, http.StatusBadRequest, "address is required")
		return
	}
	block, err := bc.MineBlock(req.Address, cfg.DefaultDifficultyBits)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// P2P yayını
	p2p.BroadcastMessage(p2p.BlockMessage(block))

	res := map[string]any{
		"success":    true,
		"reward":     blockchain.GetCurrentReward(),
		"height":     bc.GetBestHeight(),
		"block_hash": fmt.Sprintf("%x", block.Hash),
	}
	processAIBonus()
	_ = bc.SaveToFile(cfg.ChainFile)

	writeOK(w, res)
}

// --- (ZATEN) REST Transfer Handler ---
func handleSendTx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, _ := io.ReadAll(r.Body)
	var req SendRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.From == "" || req.To == "" || req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "from, to, amount required")
		return
	}

	tx, err := blockchain.NewTransaction(req.From, req.To, req.Amount, bc)
	if err != nil {
		writeOK(w, SendResponse{Success: false, Message: "create tx: " + err.Error()})
		return
	}
	if err := bc.AddTransaction(tx); err != nil {
		writeOK(w, SendResponse{Success: false, Message: "submit tx: " + err.Error()})
		return
	}

	// P2P yayını
	p2p.BroadcastMessage(p2p.TxMessage(tx))

	writeOK(w, SendResponse{
		Success: true,
		TxID:    fmt.Sprintf("%x", tx.ID),
	})
}

// --- (YENİ) Hızlı Madencilik (test/dev) ---
func handleFastMine(w http.ResponseWriter, r *http.Request) {
	// Örnek: /api/dev/fastmine?n=10&address=QC1...
	nStr := r.URL.Query().Get("n")
	addr := r.URL.Query().Get("address")

	if addr == "" {
		writeError(w, http.StatusBadRequest, "address required")
		return
	}
	n, _ := strconv.Atoi(nStr)
	if n <= 0 {
		n = 5 // varsayılan 5 blok
	}
	for i := 0; i < n; i++ {
		_, err := bc.MineBlock(addr, cfg.DefaultDifficultyBits)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	// Son bloğu yayınlamak yeterli; istersen hepsi için de yayın yapılabilir
	p2p.BroadcastMessage(p2p.BlockMessage(bc.Blocks[len(bc.Blocks)-1]))
	_ = bc.SaveToFile(cfg.ChainFile)

	writeOK(w, map[string]any{
		"success": true,
		"mined":   n,
		"height":  bc.GetBestHeight(),
	})
}

// --- AI/Bonus Otomasyonu ---
func processAIBonus() {
	fmt.Println("🔍 [AI] Bonus/Analiz sistemi başlatıldı...")
	var recentTxs []*blockchain.Transaction
	now := time.Now()
	for _, block := range bc.Blocks {
		for _, tx := range block.Transactions {
			// 24 saatlik işlemler
			if tx.Timestamp.After(now.Add(-24 * time.Hour)) {
				recentTxs = append(recentTxs, tx)
			}
		}
	}
	internal.DistributeAIBonuses(recentTxs)
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
			if tx.Sender == address {
				userTxs = append(userTxs, tx)
			}
		}
	}
	anomalies := ai.AnalyzeTransactions(userTxs, 5, 24)
	recs := ai.GenerateRecommendations(userTxs, 14, 10)
	suggestions := ai.OptimizeRewards(userTxs, 10, 1)
	result := map[string]any{
		"anomaly_report":     anomalies,
		"recommendations":    recs,
		"reward_suggestions": suggestions,
	}
	writeOK(w, result)
}

// --- GAME API ---
func handleGameScore(w http.ResponseWriter, r *http.Request) {
	player := r.URL.Query().Get("player")
	score, _ := strconv.Atoi(r.URL.Query().Get("score"))
	game.HandleTelegramScore(gameState, player, score)
	writeOK(w, map[string]any{"success": true})
}

func handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	top := game.GetTopPlayers(gameState, 10)
	writeOK(w, top)
}

// --- CTRL+C yakala ve temiz kapa ---
func trapAndShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	fmt.Println("\nShutting down...")

	// HTTP server'i nazikçe kapat
	if httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = httpServer.Shutdown(ctx)
		cancel()
	}

	// Zinciri kaydet
	if err := bc.SaveToFile(cfg.ChainFile); err != nil {
		log.Printf("save on shutdown error: %v", err)
	}
	os.Exit(0)
}
