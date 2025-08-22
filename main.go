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
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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
)

/* ====== ANSI renk sabitleri (sadece konsol görünümü için) ====== */
const (
	ansiGreen = "\x1b[32m"
	ansiCyan  = "\x1b[36m"
	ansiReset = "\x1b[0m"
)

/* ---------- API types ---------- */

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
type BurnRequest struct {
	From   string `json:"from"`
	Amount int    `json:"amount"`
}

/* ---------- Globals ---------- */

var (
	bc         *blockchain.Blockchain
	gameState  = game.NewGameState()
	cfg        *config.Config
	httpServer *http.Server
)

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

/* ---------- helpers ---------- */

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
func writeOK(w http.ResponseWriter, v interface{}) { writeJSON(w, http.StatusOK, v) }

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

/* miner address resolution: ENV -> config.json -> miner_address.txt -> generate */

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

	if exe, e := os.Executable(); e == nil {
		_ = os.Chdir(filepath.Dir(exe))
	}

	cfg, err = config.Load("config.json")
	if err != nil {
		log.Fatalf("Config yüklenemedi: %v", err)
	}

	internal.SetBonusFile(cfg.BonusFile)

	if _, err = os.Stat(cfg.ChainFile); err == nil {
		bc, err = blockchain.LoadBlockchainFromFile(cfg.ChainFile)
		if err != nil {
			log.Fatalf("Blockchain yüklenemedi: %v", err)
		}
	} else {
		bc = blockchain.NewBlockchain(cfg.InitialReward, cfg.TotalSupply)
	}

	bc.SetCoinbaseMaturity(cfg.CoinbaseMaturity)

	/* auto mode: no args -> node + api + mining */
	if len(os.Args) < 2 {
		minerAddr, _ := ensureMinerAddress()
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

		// === RENKLİ ÇIKTI (sadece bulunan bloklar yeşil) ===
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

	default:
		printUsage()
	}

	// Tek seferlik komutlar için çıkmadan kaydet
	if err := bc.SaveToFile(cfg.ChainFile); err != nil {
		log.Fatalf("Blockchain kaydedilemedi: %v", err)
	}
}

/* ---------- mining loops ---------- */

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
			p2p.BroadcastMessage(p2p.BlockMessage(blk))

			// === RENKLİ ÇIKTI (sadece bulunan bloklar yeşil) ===
			fmt.Printf(ansiGreen+"✅ Block #%d mined"+ansiReset+"  Hash: %s%s%s\n",
				blk.Index, ansiCyan, hex.EncodeToString(blk.Hash), ansiReset)

			processAIBonus()
			_ = bc.SaveToFile(cfg.ChainFile)
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

/* ---------- HTTP API ---------- */

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
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

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"service":"QuantumCoin API"}`))
	})

	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/wallet/new", handleNewWallet)
	mux.HandleFunc("/api/wallet/balance/", handleBalance)
	mux.HandleFunc("/api/mine", handleMineBlock)
	mux.HandleFunc("/api/tx/send", handleSendTx)
	mux.HandleFunc("/api/dev/fastmine", handleFastMine)

	mux.HandleFunc("/api/ai/bonus", handleAIBonus)
	mux.HandleFunc("/api/ai/analysis", handleAIAnalysis)
	mux.HandleFunc("/api/game/score", handleGameScore)
	mux.HandleFunc("/api/game/leaderboard", handleLeaderboard)

	mux.HandleFunc("/api/blocks", handleBlocksList)
	mux.HandleFunc("/api/block", handleBlockDetail)

	mux.HandleFunc("/api/tx/burn", handleBurn)
	mux.HandleFunc("/api/stake/start", handleStakeStart)
	mux.HandleFunc("/api/stake/status", handleStakeStatus)

	mux.HandleFunc("/api/mine/job", handleMineJob)
	mux.HandleFunc("/api/mine/submit", handleMineSubmit)

	addr := getHTTPAddr()
	httpServer = &http.Server{
		Addr:              addr,
		Handler:           withCORS(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	fmt.Println("HTTP API starting at http://localhost" + addr)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("http server error: %v", err)
	}
}

/* ---------- handlers ---------- */

func handleHealth(w http.ResponseWriter, _ *http.Request) {
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
	wal := wallet.NewWallet()
	writeOK(w, WalletResponse{Address: wal.GetAddress()})
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

	tx, err := blockchain.NewTransaction(req.From, req.To, req.Amount, bc)
	if err != nil {
		writeOK(w, SendResponse{Success: false, TxID: "", Message: "create tx: " + err.Error()})
		return
	}
	if err := bc.AddTransaction(tx); err != nil {
		writeOK(w, SendResponse{Success: false, TxID: "", Message: "submit tx: " + err.Error()})
		return
	}
	p2p.BroadcastMessage(p2p.TxMessage(tx))
	writeOK(w, SendResponse{Success: true, TxID: hex.EncodeToString(tx.ID)})
}

func handleFastMine(w http.ResponseWriter, r *http.Request) {
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
	p2p.BroadcastMessage(p2p.BlockMessage(bc.Blocks[len(bc.Blocks)-1]))
	_ = bc.SaveToFile(cfg.ChainFile)
	writeOK(w, map[string]any{"success": true, "mined": n, "height": bc.GetBestHeight()})
}

/* AI / Game */

func processAIBonus() {
	var recentTxs []*blockchain.Transaction
	now := time.Now()
	for _, block := range bc.Blocks {
		for _, tx := range block.Transactions {
			if tx.Timestamp.After(now.Add(-24 * time.Hour)) {
				recentTxs = append(recentTxs, tx)
			}
		}
	}
	ai.DistributeAIBonuses(recentTxs)
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
	writeOK(w, map[string]any{
		"anomaly_report":     anomalies,
		"recommendations":    recs,
		"reward_suggestions": suggestions,
	})
}

/* game mini endpoints */

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

/* explorer */

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

/* burn & stake stubs */

func handleBurn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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
	if err := bc.AddTransaction(tx); err != nil {
		writeError(w, http.StatusBadRequest, "submit tx: "+err.Error())
		return
	}
	p2p.BroadcastMessage(p2p.TxMessage(tx))
	writeOK(w, map[string]any{"success": true, "txid": hex.EncodeToString(tx.ID)})
}

func handleStakeStart(w http.ResponseWriter, _ *http.Request) {
	writeOK(w, map[string]any{"success": false, "message": "stake module coming soon"})
}
func handleStakeStatus(w http.ResponseWriter, _ *http.Request) {
	writeOK(w, map[string]any{"success": false, "message": "stake status endpoint coming soon"})
}

/* graceful shutdown */

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
