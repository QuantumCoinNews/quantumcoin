package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
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
	qconsensus "quantumcoin/consensus"
	"quantumcoin/game"
	"quantumcoin/internal"
	qnode "quantumcoin/node"
	"quantumcoin/p2p"
	"quantumcoin/security"
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

// FlexibleAmount documents the QuantumCoin component used by this module.
// İmzalı gönderim için priv_hex destekli
type FlexibleAmount string

func (v FlexibleAmount) String() string {
	return string(v)
}

// UnmarshalJSON preserves the exact user-facing QC token.
// No float64 or integer conversion occurs at the API boundary.
func parseFlexibleAmountV1(
	raw string,
) (FlexibleAmount, error) {
	value := strings.TrimSpace(raw)

	atoms, err := qconsensus.ParseQCToAtomsV1(value)
	if err != nil {
		return "", fmt.Errorf(
			"invalid QC amount: %w",
			err,
		)
	}

	if atoms <= 0 {
		return "", fmt.Errorf(
			"QC amount must be positive",
		)
	}

	return FlexibleAmount(value), nil
}

func (v *FlexibleAmount) UnmarshalJSON(data []byte) error {
	if v == nil {
		return fmt.Errorf("nil FlexibleAmount")
	}

	raw := strings.TrimSpace(string(data))
	if raw == "" {
		return fmt.Errorf("empty amount")
	}

	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}

		parsed, err := parseFlexibleAmountV1(s)
		if err != nil {
			return err
		}

		*v = parsed
		return nil
	}

	if (raw[0] < '0' || raw[0] > '9') &&
		raw[0] != '-' {
		return fmt.Errorf(
			"amount must be a JSON number or string",
		)
	}

	parsed, err := parseFlexibleAmountV1(raw)
	if err != nil {
		return err
	}

	*v = parsed
	return nil
}

type SendRequest struct {
	From    string         `json:"from"`
	To      string         `json:"to"`
	Amount  FlexibleAmount `json:"amount"`
	PrivHex string         `json:"priv_hex,omitempty"`
}

type SendResponse struct {
	Success bool   `json:"success"`
	TxID    string `json:"txid"`
	Message string `json:"message,omitempty"`
}

type BurnRequest struct {
	From    string         `json:"from"`
	PrivHex string         `json:"priv_hex,omitempty"`
	Amount  FlexibleAmount `json:"amount"`
}

// TelemetryResponse documents the QuantumCoin component used by this module.
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
	nodeRuntime     *qnode.Runtime
	bc              *blockchain.Blockchain
	gameState       = game.NewGameState()
	cfg             *config.Config
	aiAlerts        = NewAIAlertBuffer(200)
	apiReadOnlyMode bool
)

/* ---------- Mining döngüleri ---------- */

func startContinuousMining(miner string) {
	if nodeRuntime == nil {
		log.Println("mine error: node runtime unavailable")
		return
	}

	err := nodeRuntime.StartMining(
		miner,
		qnode.MiningHooksV1{
			Difficulty: currentEffectiveDifficultyBits,

			ErrorDelay: effectiveMiningDelay,

			Broadcast: func(block *blockchain.Block) {
				nodeRuntime.BroadcastP2P(
					p2p.BlockMessage(block),
				)
			},

			OnBlockMined: func(block *blockchain.Block) {
				fmt.Printf(
					ansiGreen+"✅ Block #%d mined"+ansiReset+"  Hash: %s%s%s\n",
					block.Index,
					ansiCyan,
					hex.EncodeToString(block.Hash),
					ansiReset,
				)
			},

			OnError: func(err error) {
				log.Printf(
					"mine error: %v",
					err,
				)
			},
		},
	)
	if err != nil {
		log.Printf(
			"miner start error: %v",
			err,
		)
		return
	}

	nodeRuntime.WaitMining()
}

func autosaveLoop() {
	if nodeRuntime == nil {
		log.Println(
			"autosave error: node runtime unavailable",
		)
		return
	}

	err := nodeRuntime.RunAutosave(
		context.Background(),
		10*time.Second,
		func(err error) {
			log.Println(
				"autosave error:",
				err,
			)
		},
	)
	if err != nil {
		log.Println(
			"autosave stopped:",
			err,
		)
	}
}

func reloadAPIChainFromDiskIfChanged() {
	if nodeRuntime == nil {
		return
	}

	nodeRuntime.SetAPIReadOnly(apiReadOnlyMode)

	reloaded, err := nodeRuntime.ReloadChainFromDiskIfChanged()
	if err != nil {
		log.Printf(
			"api live reload error: %v",
			err,
		)
		return
	}

	if !reloaded {
		return
	}

	loadedBC, _ := nodeRuntime.Snapshot()
	if loadedBC == nil {
		log.Print(
			"api live reload returned nil blockchain",
		)
		return
	}

	bc = loadedBC

	log.Printf(
		"api live reload: chain height=%d file=%s",
		bc.BestHeightSnapshot(),
		cfg.ChainFile,
	)
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
	jobMu  sync.Mutex
	curJob *webJob
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
	payload, err := json.Marshal(v)
	if err != nil {
		log.Printf("http: JSON response encode failed: %v", err)
		status = http.StatusInternalServerError
		payload = []byte(`{"success":false,"message":"failed to encode response"}`)
	}

	payload = append(payload, '\n')

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if _, err := w.Write(payload); err != nil {
		log.Printf("http: JSON response write failed: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"success": false, "message": msg})
}

func getHTTPPort() string {
	if cfg == nil {
		return ""
	}

	port, err := config.ResolveEffectiveHTTPPortV1(cfg)
	if err != nil {
		log.Printf(
			"[config] HTTP port policy rejected legacy override: %v; using canonical=%s",
			err,
			cfg.HTTPPort,
		)

		return cfg.HTTPPort
	}

	return port
}

func getHTTPAddr() string {
	addr := getHTTPPort()
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}

	return addr
}

// getAPIListenAddr preserves the localhost default while permitting
// an explicit host/interface only when it uses the canonical API port.
func getAPIListenAddr() string {
	canonicalAddr := getHTTPAddr()
	if canonicalAddr == "" {
		return ""
	}

	canonicalPort := strings.TrimPrefix(
		canonicalAddr,
		":",
	)

	localDefault := "127.0.0.1:" + canonicalPort

	override := strings.TrimSpace(
		os.Getenv("QC_API_LISTEN"),
	)

	if override == "" {
		return localDefault
	}

	host, port, err := net.SplitHostPort(override)
	if err != nil {
		log.Printf(
			"[config] invalid QC_API_LISTEN=%q: %v; using %s",
			override,
			err,
			localDefault,
		)

		return localDefault
	}

	if port != canonicalPort {
		log.Printf(
			"[config] QC_API_LISTEN=%q conflicts with canonical port %s; using %s",
			override,
			canonicalPort,
			localDefault,
		)

		return localDefault
	}

	if strings.TrimSpace(host) == "" {
		return localDefault
	}

	return override
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

	// Canonical node bootstrap.
	nodeRuntime, err = qnode.Bootstrap(
		qnode.BootstrapOptions{
			ConfigPath: "config.json",
			StrictChain: strings.TrimSpace(
				os.Getenv("QC_STRICT_CHAIN"),
			) == "1",
		},
	)
	if err != nil {
		log.Fatalf(
			"Node bootstrap failed: %v",
			err,
		)
	}

	// Compatibility bridge:
	// Existing API, P2P, miner, Telegram Game and release paths
	// continue using the historical globals during runtime migration migration.
	bc, cfg = nodeRuntime.Snapshot()

	if bc == nil || cfg == nil {
		log.Fatal(
			"Node bootstrap returned incomplete runtime state",
		)
	}

	// Auto mode: arg yoksa node + API + mining
	if len(os.Args) < 2 {
		minerAddr := getDefaultAddress()
		fmt.Printf("⛏️  Auto mode: node+api+mining -> %s (difficulty=%d)\n", minerAddr, currentEffectiveDifficultyBits())
		go startHTTPAPI()
		go autosaveLoop()
		go startContinuousMining(minerAddr)
		go trapAndShutdown()
		p := strings.TrimPrefix(cfg.P2PPort, ":")
		if err := nodeRuntime.ServeP2P(p); err != nil {
			log.Printf("p2p server error: %v", err)
		}
		return
	}

	switch os.Args[1] {
	case "run":
		if len(os.Args) >= 3 {
			port := os.Args[2]
			go startHTTPAPI()
			go autosaveLoop()
			go trapAndShutdown()
			if err := nodeRuntime.ServeP2P(port); err != nil {
				log.Printf("p2p server error: %v", err)
			}
		} else {
			go startHTTPAPI()
			go autosaveLoop()
			go trapAndShutdown()
			p := strings.TrimPrefix(cfg.P2PPort, ":")
			if err := nodeRuntime.ServeP2P(p); err != nil {
				log.Printf("p2p server error: %v", err)
			}
		}

	case "run-mine":
		if len(os.Args) < 3 {
			fmt.Println("Usage: run-mine [miner]")
			return
		}
		miner := os.Args[2]
		go startHTTPAPI()
		go autosaveLoop()
		go startContinuousMining(miner)
		go trapAndShutdown()
		p := strings.TrimPrefix(cfg.P2PPort, ":")
		if err := nodeRuntime.ServeP2P(p); err != nil {
			log.Printf("p2p server error: %v", err)
		}

	case "api":
		apiReadOnlyMode = true
		// API mode is read-only for chain file: no autosave and no final save.
		// Read endpoints reload chain_data.dat when miner writes new blocks.
		go trapAPIShutdown()
		startHTTPAPI()
		return

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
		if err := nodeRuntime.ConnectPeer(port, address); err != nil {
			log.Printf("p2p connect error: %v", err)
		}

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
		nodeRuntime.BroadcastP2P(p2p.TxMessage(tx))
		fmt.Printf("✓ Transaction accepted and broadcasted (txid=%s)\n", hex.EncodeToString(tx.ID))

	case "mine":
		if len(os.Args) < 3 {
			fmt.Println("Usage: mine [miner]")
			return
		}

		miner := os.Args[2]
		difficulty := currentEffectiveDifficultyBits()

		candidateHeight := bc.BestHeightSnapshot() + 1
		reward := bc.ExpectedSubsidyForHeightSnapshot(
			candidateHeight,
		)

		block, err := nodeRuntime.MineOnceV1(
			miner,
			difficulty,
		)
		if err != nil {
			log.Println("mining failed:", err)
			return
		}

		nodeRuntime.BroadcastP2P(
			p2p.BlockMessage(block),
		)

		fmt.Printf(
			ansiGreen+"✅ New block mined by %s"+ansiReset+"\n",
			miner,
		)

		fmt.Printf(
			"   Hash:   %s%s%s\n",
			ansiGreen,
			hex.EncodeToString(block.Hash),
			ansiReset,
		)

		fmt.Printf(
			"   Height: %d  Reward: %d QC\n",
			block.Index,
			reward,
		)

		// processAIBonus() disabled:
		// AI monitors only, no mining bonus / no supply effect.
		_ = bc.SaveToFile(cfg.ChainFile)

	case "mine-forever":
		if len(os.Args) < 3 {
			fmt.Println("Usage: mine-forever [miner]")
			return
		}
		miner := os.Args[2]
		go autosaveLoop()
		go trapAndShutdown()
		startContinuousMining(miner)

	case "print":
		for _, block := range bc.AllBlocksSnapshot() {
			fmt.Printf("📦 Block #%d\n", block.Index)
			fmt.Printf("⛏️  Miner     : %s\n", block.Miner)
			fmt.Printf("🧱 Hash       : %s\n", hex.EncodeToString(block.Hash))
			fmt.Printf("🔗 PrevHash   : %s\n", hex.EncodeToString(block.PrevHash))
			fmt.Println("📝 Transactions:")
			for _, tx := range block.Transactions {
				fmt.Printf("  TxID: %s\n", hex.EncodeToString(tx.ID))
				for _, out := range tx.Outputs {
					fmt.Printf("    🔸 Amount: %s QC\n", bc.OutputAmountQCAtHeightV1(out.Amount, block.Index))
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

// protected, handler'ı zorunlu header token kontrolü ve body limitiyle sarar.
// QC_API_TOKEN her çalışma modunda zorunludur.
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

/* ---------- HTTP API ---------- */

// main.go içinde, HTTP API bölümünde
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corsOK := applyCORSHeadersV1(w, r)
		originPresent := strings.TrimSpace(r.Header.Get("Origin")) != ""

		if r.Method == http.MethodOptions {
			if !corsOK {
				writeError(w, http.StatusForbidden, "cors origin not allowed")
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Public browser-originated requests must fail closed before any
		// application handler can perform reads or side effects.
		// Requests without Origin remain available to CLI/server clients.
		if originPresent && !corsOK {
			writeError(w, http.StatusForbidden, "cors origin not allowed")
			return
		}

		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func registerAPINotFoundBoundary(mux *http.ServeMux) {
	if mux == nil {
		return
	}

	notFound := func(w http.ResponseWriter, _ *http.Request) {
		writeError(
			w,
			http.StatusNotFound,
			"api endpoint not found",
		)
	}

	mux.HandleFunc("/api", notFound)
	mux.HandleFunc("/api/", notFound)
}

func shutdownAPIHTTPServer(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("API shutdown context is nil")
	}

	if nodeRuntime == nil {
		return fmt.Errorf("node runtime unavailable")
	}

	server := nodeRuntime.HTTP()
	if server == nil {
		return fmt.Errorf("API HTTP server unavailable")
	}

	return server.Shutdown(ctx)
}

func trapAPIShutdown() {
	signalChannel := make(chan os.Signal, 1)

	signal.Notify(
		signalChannel,
		os.Interrupt,
	)
	defer signal.Stop(signalChannel)

	<-signalChannel

	fmt.Println()
	fmt.Println("Shutting down API...")

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := shutdownAPIHTTPServer(ctx); err != nil {
		log.Printf(
			"API graceful shutdown error: %v",
			err,
		)
	}
}
func newHTTPAPIHandlerV1(
	guard *HTTPGuard,
	crit *EndpointLimiter,
) http.Handler {
	mux := http.NewServeMux()

	if crit == nil {
		crit = NewEndpointLimiter()
	}

	// Temel API uçları
	mux.HandleFunc("/api/health", protected(handleHealth))
	mux.HandleFunc("/api/network/label", handleNetworkLabelV1)
	mux.HandleFunc("/api/telemetry", protected(handleTelemetry))
	mux.HandleFunc("/api/economy/policy", protected(handleEconomyPolicy))
	mux.HandleFunc("/api/nft/monthly/program", protected(func(w http.ResponseWriter, r *http.Request) {
		if nodeRuntime == nil {
			writeError(w, http.StatusServiceUnavailable, "node runtime unavailable")
			return
		}
		nodeRuntime.ServeNFTGameProgramV1(w, r)
	}))
	mux.HandleFunc(
		"/api/nft/monthly/mint",
		crit.WrapFunc("nft_monthly_mint", 10*time.Second, 5, protected(func(w http.ResponseWriter, r *http.Request) {
			if nodeRuntime == nil {
				writeError(w, http.StatusServiceUnavailable, "node runtime unavailable")
				return
			}
			nodeRuntime.ServeNFTMonthlyMintV1(w, r)
		})),
	)
	mux.HandleFunc(
		"/api/nft/monthly/receipt",
		crit.WrapFunc("nft_monthly_receipt", 10*time.Second, 10, protected(func(w http.ResponseWriter, r *http.Request) {
			if nodeRuntime == nil {
				writeError(w, http.StatusServiceUnavailable, "node runtime unavailable")
				return
			}
			nodeRuntime.ServeNFTMonthlyReceiptV1(w, r)
		})),
	)
	mux.HandleFunc("/api/security/summary", protected(handleSecuritySummary))
	mux.HandleFunc("/api/security/events", protected(handleSecurityEvents))
	mux.HandleFunc("/api/security/score", protected(handleSecurityScore))
	mux.HandleFunc("/api/security/node-health", protected(handleNodeHealth))
	mux.HandleFunc("/api/security/peer-reputation", protected(handlePeerReputation))
	mux.HandleFunc("/api/security/dashboard", protected(handleSecurityDashboard))
	mux.HandleFunc("/api/security/admin-report", protected(handleSecurityAdminReport))
	mux.HandleFunc("/api/security/wallet-anomaly", protected(handleWalletAnomaly))
	mux.HandleFunc("/api/security/mempool-abuse", protected(handleMempoolAbuse))
	mux.HandleFunc("/api/security/anti-spam-fee-policy", protected(handleAntiSpamFeePolicy))
	mux.HandleFunc("/api/security/finality-checkpoint", protected(handleFinalityCheckpoint))
	mux.HandleFunc("/api/wallet/new", protected(handleNewWallet))
	mux.HandleFunc("/api/wallet/address", protected(handleWalletAddress))
	mux.HandleFunc("/api/wallet/balance/", protected(handleBalance))
	mux.HandleFunc("/api/mine", protected(handleMineBlock))

	// Kritik uç: tx/send – 10 saniyede en fazla 5 istek.
	mux.HandleFunc(
		"/api/tx/send",
		crit.WrapFunc(
			"tx_send",
			10*time.Second,
			5,
			protected(handleSendTx),
		),
	)
	mux.HandleFunc("/api/tx/status", protected(handleTxStatus))
	mux.HandleFunc("/api/dev/fastmine", protected(handleFastMine))

	// AI ve oyun.
	mux.HandleFunc("/api/ai/alerts", protected(handleAIAlerts))
	mux.HandleFunc("/api/ai/bonus", protected(handleAIBonus))
	mux.HandleFunc("/api/ai/analysis", protected(handleAIAnalysis))
	mux.HandleFunc("/api/ai/security-dashboard", protected(handleAISecurityDashboard))
	mux.HandleFunc("/api/game/score", protected(handleGameScore))
	mux.HandleFunc("/api/game/leaderboard", protected(handleLeaderboard))

	// Explorer.
	mux.HandleFunc("/api/blocks", protected(handleBlocksList))
	mux.HandleFunc("/api/block", protected(handleBlockDetail))

	// Burn ve stake.
	mux.HandleFunc(
		"/api/tx/burn",
		crit.WrapFunc(
			"tx_burn",
			10*time.Second,
			5,
			protected(handleBurn),
		),
	)
	mux.HandleFunc("/api/stake/start", protected(handleStakeStart))
	mux.HandleFunc("/api/stake/status", protected(handleStakeStatus))

	// Web miner.
	mux.HandleFunc("/api/mine/job", protected(handleMineJob))
	mux.HandleFunc(
		"/api/mine/submit",
		crit.WrapFunc(
			"mine_submit",
			10*time.Second,
			20,
			protected(handleMineSubmit),
		),
	)

	// Miner kontrol.
	mux.HandleFunc(
		"/api/miner/start",
		crit.WrapFunc(
			"miner_start",
			10*time.Second,
			3,
			protected(handleMinerStart),
		),
	)
	mux.HandleFunc(
		"/api/miner/stop",
		crit.WrapFunc(
			"miner_stop",
			10*time.Second,
			3,
			protected(handleMinerStop),
		),
	)
	mux.HandleFunc("/api/miner/status", protected(handleMinerStatus))

	// Bilinmeyen API yolları hiçbir zaman SPA index'e düşmemeli.
	registerAPINotFoundBoundary(mux)

	// Web cüzdan SPA'sı public kalır.
	if h, err := webui.Handler(); err == nil {
		mux.Handle("/", h)
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			writeJSON(
				w,
				http.StatusOK,
				map[string]any{
					"ok":      true,
					"service": "QuantumCoin API",
				},
			)
		})
	}

	var handler http.Handler = withCORS(mux)
	if guard != nil {
		handler = guard.Wrap(handler)
	}

	return handler
}

func startHTTPAPI() {
	listenAddr := getAPIListenAddr()
	handler := newHTTPAPIHandlerV1(
		defaultHTTPGuard,
		criticalLimiter,
	)

	fmt.Println("HTTP API starting at http://" + listenAddr)

	if err := nodeRuntime.ServeHTTP(
		listenAddr,
		handler,
		qnode.DefaultHTTPServerOptionsV1(),
	); err != nil {
		log.Printf(
			"http server error: %v",
			err,
		)
	}
}

// ECONOMY_POLICY_V1
// GET /api/economy/policy
// Read-only economy policy endpoint.
// Safe boundary: no mint, no transfer, no balance change.
func handleEconomyPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, blockchain.EconomyPolicySummary())
}

// SECURITY_EVENT_LOG_V1
// GET /api/security/summary
// Read-only node security summary.
func handleSecuritySummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, security.Summary("."))
}

// SECURITY_EVENT_LOG_V1
// GET /api/security/events?limit=100
// Read-only local security events.
func handleSecurityEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	writeJSON(w, http.StatusOK, security.ReadEvents(".", limit))
}

/* ---------- Handler yardımcıları ---------- */

// Eğer yukarıda zaten writeOK tanımlıysa, bunu SİLEBİLİRSİN (tek tanım kalsın).
func writeOK(w http.ResponseWriter, v interface{}) {
	writeJSON(w, http.StatusOK, v)
}

// ----- Zorunlu header tabanlı API token kontrolü -----
//
// Korunan API uçlarında QC_API_TOKEN her çalışma modunda zorunludur.
// İstek tokenı yalnızca X-QC-API-KEY veya Authorization: Bearer ile taşınır.
// Query ve form/body token aktarımı hiçbir modda kabul edilmez.
// PUBLIC_TESTNET_API_TOKEN_REQUIRED_V1
//
// Local/dev mode keeps the old behavior: if QC_API_TOKEN is empty, protected
// endpoints are allowed for local development.
// Public testnet mode is stricter: when QC_PUBLIC_TESTNET=1, or when a
// testnet node is explicitly exposed through QC_API_LISTEN, QC_API_TOKEN must
// be configured.
func apiEnvBool(name string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func apiEnvContainsTestnet(name string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(os.Getenv(name))), "testnet")
}

func apiListenAddrIsPublic() bool {
	listen := strings.ToLower(strings.TrimSpace(os.Getenv("QC_API_LISTEN")))
	if listen == "" {
		return false
	}

	host := listen
	if strings.Contains(host, "://") {
		host = strings.TrimPrefix(host, "http://")
		host = strings.TrimPrefix(host, "https://")
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[] ")

	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		return true
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	return !ip.IsLoopback()
}

func isPublicTestnetAPITokenRequired() bool {
	if apiEnvBool("QC_PUBLIC_TESTNET") ||
		apiEnvBool("QC_API_PUBLIC_TESTNET") ||
		apiEnvBool("QC_TESTNET_PUBLIC_API") {
		return true
	}

	testnetMode :=
		apiEnvBool("QC_TESTNET") ||
			apiEnvContainsTestnet("QC_NET_MAGIC") ||
			apiEnvContainsTestnet("QC_NETWORK") ||
			apiEnvContainsTestnet("QC_NETWORK_MODE") ||
			apiEnvContainsTestnet("QC_CHAIN_MODE") ||
			apiEnvContainsTestnet("QC_MODE")

	return testnetMode && apiListenAddrIsPublic()
}
func checkAPIToken(w http.ResponseWriter, r *http.Request) bool {
	expected := strings.TrimSpace(os.Getenv("QC_API_TOKEN"))
	if expected == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="QuantumCoin API"`)
		writeError(w, http.StatusUnauthorized, "missing QC_API_TOKEN configuration")
		return false
	}

	// API tokens are accepted only through request headers in every mode.
	// URL query and form/body transport can expose credentials through logs,
	// browser history, reverse proxies, analytics, and referrer propagation.
	token := strings.TrimSpace(r.Header.Get("X-QC-API-KEY"))

	if token == "" {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			token = strings.TrimSpace(auth[7:])
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
	// Health verisi hassas durum bilgisi içerdiği için route düzeyinde korunur.
	writeOK(w, map[string]any{
		"ok":       true,
		"height":   bc.BestHeightSnapshot(),
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

	address := strings.TrimSpace(getDefaultAddress())
	if address == "" || !wallet.ValidateAddress(address) {
		writeError(
			w,
			http.StatusServiceUnavailable,
			"wallet address unavailable",
		)
		return
	}

	writeOK(w, WalletResponse{Address: address})
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

	address := strings.TrimSpace(parts[4])
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

	totalQC, spendableQC, height :=
		chain.BalanceQCSnapshotV1(address)

	writeOK(w, BalanceResponse{
		Balance:   totalQC,
		Spendable: spendableQC,
		Height:    height,
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
		if writeRequestBodyTooLargeV1(w, err) {
			return
		}

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
	candidateHeight := bc.BestHeightSnapshot() + 1
	reward := bc.ExpectedSubsidyForHeightSnapshot(
		candidateHeight,
	)

	block, err := nodeRuntime.MineOnceV1(
		req.Address,
		currentEffectiveDifficultyBits(),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	nodeRuntime.BroadcastP2P(p2p.BlockMessage(block))

	writeOK(w, map[string]any{
		"success":    true,
		"reward":     reward,
		"height":     bc.BestHeightSnapshot(),
		"block_hash": hex.EncodeToString(block.Hash),
	})

	// processAIBonus() disabled: AI monitors only, no mining bonus / no supply effect.
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
		if writeRequestBodyTooLargeV1(w, err) {
			return
		}

		writeError(w, http.StatusBadRequest, "read body: "+err.Error())
		return
	}

	var req SendRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.From == "" || req.To == "" || strings.TrimSpace(req.Amount.String()) == "" {
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

	tx, err := blockchain.NewTransactionFromQCAmountV1(req.From, req.To, req.Amount.String(), bc)
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

	// --- ÇEKİRDEK BLOCKCHAIN MEMPOOL'U ---
	if err := bc.AddTransaction(tx); err != nil {
		writeOK(w, SendResponse{
			Success: false,
			TxID:    "",
			Message: "submit tx: " + err.Error(),
		})
		return
	}

	nodeRuntime.BroadcastP2P(p2p.TxMessage(tx))

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
	// QC16: fastmine yalnızca local/dev modda açık; public testnette kapalı.
	if !requireDevMinerEndpointAllowedV1(w, r, "/api/dev/fastmine") {
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
		if _, err := nodeRuntime.MineOnceV1(addr, currentEffectiveDifficultyBits()); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	// Son bloğu peer’lara yay
	if bc.BlockCountSnapshot() > 0 {
		last := bc.TipSnapshot()
		nodeRuntime.BroadcastP2P(p2p.BlockMessage(last))
	}

	_ = bc.SaveToFile(cfg.ChainFile)

	writeOK(w, map[string]any{
		"success": true,
		"mined":   n,
		"height":  bc.BestHeightSnapshot(),
	})
}

// /api/telemetry
func handleTelemetry(w http.ResponseWriter, r *http.Request) {
	reloadAPIChainFromDiskIfChanged()
	// Telemetry read-only, token şart değil – istersen:
	// if !checkAPIToken(w, r) { return }

	height := -1
	blockCount := 0
	totalSupply := 0
	currentReward := 0

	if bc != nil {
		height = bc.BestHeightSnapshot()
		blockCount = bc.BlockCountSnapshot()
		totalSupply = bc.TotalSupplySnapshot()
		nextHeight := height + 1
		currentReward = bc.ExpectedSubsidyForHeightSnapshot(
			nextHeight,
		)
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	writeOK(w, TelemetryResponse{
		Height:        height,
		BlockCount:    blockCount,
		Peers:         nodeRuntime.PeerCount(),
		MinerRunning:  nodeRuntime != nil && nodeRuntime.IsMining(),
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

// NODE_HEALTH_SUMMARY_V1
// NODE_SYNC_DRIFT_DETECTOR_V1
// GET /api/security/node-health
func handleNodeHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	reloadAPIChainFromDiskIfChanged()

	height := -1
	blockCount := 0
	totalSupply := 0
	lastBlockAgeSeconds := int64(-1)

	if bc != nil {
		height = bc.BestHeightSnapshot()
		blockCount = bc.BlockCountSnapshot()
		totalSupply = bc.TotalSupplySnapshot()

		if last := bc.TipSnapshot(); last != nil {
			lastBlockAgeSeconds = time.Now().Unix() - last.Timestamp
		}
	}

	mempoolSize := bc.PendingTransactionCountSnapshot()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	peerCount := nodeRuntime.PeerCount()
	goRoutines := runtime.NumGoroutine()

	health := "GOOD"
	syncStatus := "SYNC_OK"
	driftWarning := false

	if peerCount == 0 {
		health = "ISOLATED"
		syncStatus = "NO_PEERS_LOCAL_ONLY"
	}

	if lastBlockAgeSeconds >= 0 && lastBlockAgeSeconds > int64(6*time.Hour/time.Second) {
		health = "WATCH"
		syncStatus = "NODE_STALE_BLOCKS"
		driftWarning = true

		_ = security.AppendEvent(".", security.NewEvent(
			"NODE_STALE_BLOCKS",
			security.SeverityWarning,
			"node_health",
			"",
			"Last block is older than expected threshold.",
			map[string]any{
				"lastBlockAgeSeconds": lastBlockAgeSeconds,
				"height":              height,
				"peerCount":           peerCount,
			},
		))
	}

	if goRoutines > 500 {
		health = "WATCH"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"version":     "NODE_HEALTH_SUMMARY_V1",
		"syncVersion": "NODE_SYNC_DRIFT_DETECTOR_V1",
		"health":      health,

		"height":              height,
		"blockCount":          blockCount,
		"totalSupply":         totalSupply,
		"peerCount":           peerCount,
		"mempoolSize":         mempoolSize,
		"lastBlockAgeSeconds": lastBlockAgeSeconds,
		"syncStatus":          syncStatus,
		"driftWarning":        driftWarning,

		"cpuCount":   runtime.NumCPU(),
		"goRoutines": goRoutines,
		"memoryMB":   float64(m.Alloc) / (1024 * 1024),

		"minerRunning": nodeRuntime != nil && nodeRuntime.IsMining(),

		"safeBoundary": map[string]bool{
			"changesChain":       false,
			"changesBalances":    false,
			"changesConsensus":   false,
			"changesTotalSupply": false,
		},
	})
}

/* ---------- AI & Game ---------- */

func effectiveMiningDelay(defaultDelay time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv("QC_MINING_DELAY_MS"))
	if v == "" {
		return defaultDelay
	}

	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil || n < 0 || n > 60000 {
		log.Printf("[config] invalid QC_MINING_DELAY_MS=%q; using default=%s", v, defaultDelay)
		return defaultDelay
	}

	return time.Duration(n) * time.Millisecond
}
func effectiveDifficultyBits(dynamicBits int) int {
	info, err := config.ResolveEffectiveDifficultyBitsV1(
		cfg,
		dynamicBits,
	)
	if err != nil {
		log.Printf(
			"[config] effective difficulty policy rejected override: %v; using dynamic=%d",
			err,
			dynamicBits,
		)

		return dynamicBits
	}

	if info.OverrideApplied {
		fmt.Printf(
			"[config] %s override: %d (dynamic=%d, profile=%s)\n",
			info.OverrideSource,
			info.EffectiveBits,
			info.DynamicBits,
			info.Profile,
		)
	}

	return info.EffectiveBits
}

func currentEffectiveDifficultyBits() int {
	if cfg == nil {
		return 0
	}

	dynamicBits := cfg.DefaultDifficultyBits

	if bc != nil {
		dynamicBits = bc.NextDifficultySnapshot(
			cfg.DefaultDifficultyBits,
		)
	}

	return effectiveDifficultyBits(dynamicBits)
}

//lint:ignore U1000 retained disabled AI bonus stub; AI must not affect consensus or supply
func processAIBonus() {
	// AI bonus is disabled for mining consensus and supply safety.
	// AI can monitor/analyze only; it must not mint, write bonus files, or alter balances.
}

func handleAIBonus(w http.ResponseWriter, r *http.Request) {
	if r == nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	address := r.URL.Query().Get("address")
	bonuses := internal.ListBonuses(address)
	writeOK(w, bonuses)
}

func handleAIAnalysis(w http.ResponseWriter, r *http.Request) {
	if r == nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	address := r.URL.Query().Get("address")

	lite := make([]ai.TxLite, 0)
	for _, block := range bc.AllBlocksSnapshot() {
		if block == nil {
			continue
		}
		var userTxs []*blockchain.Transaction
		for _, tx := range block.Transactions {
			if tx != nil && (address == "" || tx.Sender == address) {
				userTxs = append(userTxs, tx)
			}
		}
		lite = append(lite, bc.ToAITxLiteAtHeightV1(userTxs, block.Index)...)
	}

	if !ai.Enabled() {
		writeOK(w, map[string]any{
			"anomaly_report":     []any{},
			"recommendations":    []any{},
			"reward_suggestions": []any{},
		})
		return
	}

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

const (
	defaultBlocksPageLimit = 20
	maxBlocksPageLimit     = 200
)

func parseBlocksPagination(
	r *http.Request,
) (limit int, offset int) {
	limit = defaultBlocksPageLimit
	offset = 0

	if r == nil {
		return limit, offset
	}

	query := r.URL.Query()

	if raw :=
		strings.TrimSpace(
			query.Get("limit"),
		); raw != "" {
		if parsed, err :=
			strconv.Atoi(raw); err == nil &&
			parsed > 0 {
			limit = parsed
		}
	}

	if limit > maxBlocksPageLimit {
		limit = maxBlocksPageLimit
	}

	if raw :=
		strings.TrimSpace(
			query.Get("offset"),
		); raw != "" {
		if parsed, err :=
			strconv.Atoi(raw); err == nil &&
			parsed >= 0 {
			offset = parsed
		}
	}

	return limit, offset
}
func handleBlocksList(w http.ResponseWriter, r *http.Request) {
	limit, offset :=
		parseBlocksPagination(
			r,
		)

	blocks, total :=
		bc.BlocksPageFromTipSnapshot(
			offset,
			limit,
		)

	w.Header().Set(
		"X-Total-Count",
		strconv.Itoa(total),
	)
	w.Header().Set(
		"X-Page-Limit",
		strconv.Itoa(limit),
	)
	w.Header().Set(
		"X-Page-Offset",
		strconv.Itoa(offset),
	)

	nextOffset :=
		offset + len(blocks)

	hasMore :=
		nextOffset < total

	w.Header().Set(
		"X-Has-More",
		strconv.FormatBool(hasMore),
	)

	if hasMore {
		w.Header().Set(
			"X-Next-Offset",
			strconv.Itoa(nextOffset),
		)
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

	summaries :=
		make(
			[]bsum,
			0,
			len(blocks),
		)

	for _, b := range blocks {
		summaries = append(
			summaries,
			bsum{
				Index:      b.Index,
				Hash:       hex.EncodeToString(b.Hash),
				PrevHash:   hex.EncodeToString(b.PrevHash),
				Timestamp:  b.Timestamp,
				Miner:      b.Miner,
				Difficulty: b.Difficulty,
				TxCount:    len(b.Transactions),
			},
		)
	}

	writeOK(w, summaries)
}

func handleBlockDetail(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if idxStr := q.Get("index"); idxStr != "" {
		if idx, err := strconv.Atoi(idxStr); err == nil {
			if blk := bc.BlockByIndexSnapshot(idx); blk != nil {
				writeOK(w, newBlockDetailPresentationV1(bc, blk))
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
		if blk := bc.BlockByHashSnapshot(raw); blk != nil {
			writeOK(w, newBlockDetailPresentationV1(bc, blk))
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

	// API token kontrolü protected(...) wrapper'ında yapılır.

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		if writeRequestBodyTooLargeV1(w, err) {
			return
		}

		writeError(w, http.StatusBadRequest, "read body: "+err.Error())
		return
	}

	var req BurnRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.From == "" ||
		strings.TrimSpace(req.Amount.String()) == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"from and positive amount required",
		)
		return
	}

	privHex := strings.TrimSpace(req.PrivHex)
	if privHex == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"missing priv_hex for signing",
		)
		return
	}

	tx, err :=
		blockchain.NewCanonicalBurnTransactionFromQCAmountV1(
			req.From,
			req.Amount.String(),
			bc,
		)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"create burn tx: "+err.Error(),
		)
		return
	}

	priv, err := wallet.ImportPrivateKeyHex(privHex)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid priv_hex: "+err.Error(),
		)
		return
	}

	if err := tx.Sign(priv); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"sign burn tx: "+err.Error(),
		)
		return
	}

	if !tx.Verify() {
		writeError(
			w,
			http.StatusBadRequest,
			"verify failed",
		)
		return
	}

	if err := bc.AddTransaction(tx); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"submit tx: "+err.Error(),
		)
		return
	}

	nodeRuntime.BroadcastP2P(p2p.TxMessage(tx))

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

	writeError(w, http.StatusNotImplemented, "stake module coming soon")
}

func handleStakeStatus(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotImplemented, "stake status endpoint coming soon")
}

/* ---------- Graceful shutdown ---------- */

func trapAndShutdown() {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(
		signalChannel,
		os.Interrupt,
	)

	<-signalChannel

	fmt.Println()
	fmt.Println("Shutting down...")

	if nodeRuntime == nil {
		log.Print(
			"shutdown error: node runtime unavailable",
		)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		3*time.Second,
	)
	defer cancel()

	if err := nodeRuntime.Shutdown(ctx); err != nil {
		log.Printf(
			"shutdown error: %v",
			err,
		)
		os.Exit(1)
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
	last := bc.TipSnapshot()
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
		Height:     bc.BestHeightSnapshot(),
		ExpiresAt:  time.Now().Add(60 * time.Second),
	}
}

func handleMineJob(w http.ResponseWriter, r *http.Request) {
	addr := r.URL.Query().Get("address")
	if addr == "" {
		writeError(w, http.StatusBadRequest, "address required")
		return
	}
	diff := currentEffectiveDifficultyBits()
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
		if writeRequestBodyTooLargeV1(w, err) {
			return
		}

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
		if blk, err := nodeRuntime.MineOnceV1(miner, currentEffectiveDifficultyBits()); err == nil {
			nodeRuntime.BroadcastP2P(p2p.BlockMessage(blk))
			// processAIBonus() disabled: AI monitors only, no mining bonus / no supply effect.
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

	if !requireDevMinerEndpointAllowedV1(w, r, "/api/miner/start") {
		return
	}
	var addr string
	if r.Method == http.MethodPost {
		defer r.Body.Close()
		var req minerStartReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if writeRequestBodyTooLargeV1(w, err) {
				return
			}
		}
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

	if nodeRuntime != nil && nodeRuntime.IsMining() {
		writeOK(w, map[string]any{
			"running":    true,
			"message":    "miner already running",
			"address":    addr,
			"difficulty": currentEffectiveDifficultyBits(),
			"height":     bc.BestHeightSnapshot(),
		})
		return
	}

	go startContinuousMining(addr)

	writeOK(w, map[string]any{
		"running":    true,
		"address":    addr,
		"difficulty": currentEffectiveDifficultyBits(),
		"height":     bc.BestHeightSnapshot(),
	})
}

func handleMinerStop(w http.ResponseWriter, r *http.Request) {
	// --- Güvenlik: Miner'ı uzaktan durdurmak için token iste ---
	if !checkAPIToken(w, r) {
		return
	}

	if !requireDevMinerEndpointAllowedV1(w, r, "/api/miner/stop") {
		return
	}

	if nodeRuntime == nil ||
		!nodeRuntime.IsMining() {
		writeOK(w, map[string]any{
			"running": false,
			"message": "miner not running",
			"height":  bc.BestHeightSnapshot(),
		})
		return
	}

	if !nodeRuntime.StopMining() {
		writeOK(w, map[string]any{
			"running": true,
			"message": "miner stop already requested",
			"height":  bc.BestHeightSnapshot(),
		})
		return
	}

	writeOK(w, map[string]any{
		"running": false,
		"message": "miner stop requested",
		"height":  bc.BestHeightSnapshot(),
	})
}

func handleMinerStatus(w http.ResponseWriter, r *http.Request) {
	if !requireDevMinerEndpointAllowedV1(w, r, "/api/miner/status") {
		return
	}
	writeOK(w, map[string]any{
		"running":  nodeRuntime != nil && nodeRuntime.IsMining(),
		"height":   bc.BestHeightSnapshot(),
		"bits":     currentEffectiveDifficultyBits(),
		"httpPort": getHTTPPort(),
	})
}

// PEER_REPUTATION_API_V1
// GET /api/security/peer-reputation
func handlePeerReputation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if !nodeRuntime.PeerSecurityAvailable() {
		writeJSON(w, http.StatusOK, map[string]any{
			"version": "PEER_REPUTATION_API_V1",
			"peers":   map[string]any{},
			"status":  "NO_PEER_SECURITY",
		})
		return
	}

	writeJSON(w, http.StatusOK, nodeRuntime.PeerReputationSnapshot())
}

// SECURITY_SCORE_ENGINE_V1
// GET /api/security/score
func handleSecurityScore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, security.Score("."))
}

// SECURITY_DASHBOARD_API_V1
// GET /api/security/dashboard
func handleSecurityDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	peerReputation := map[string]any{
		"version": "PEER_REPUTATION_ENGINE_V1",
		"peers":   map[string]any{},
		"status":  "NO_PEER_SECURITY",
	}
	if nodeRuntime.PeerSecurityAvailable() {
		peerReputation = nodeRuntime.PeerReputationSnapshot()
	}

	walletObservations := make([]security.WalletTxObservation, 0)
	for _, block := range bc.AllBlocksSnapshot() {
		if block == nil {
			continue
		}
		for _, tx := range block.Transactions {
			if tx == nil {
				continue
			}

			txid := ""
			if len(tx.ID) > 0 {
				txid = fmt.Sprintf("%x", tx.ID)
			}

			walletObservations = append(walletObservations, security.WalletTxObservation{
				TxID:       txid,
				Sender:     tx.Sender,
				Amount:     int(bc.TransactionAmountQCAtHeightV1(tx.Amount, block.Index)),
				AmountQC:   bc.TransactionAmountQCAtHeightV1(tx.Amount, block.Index),
				Inputs:     len(tx.Inputs),
				Outputs:    len(tx.Outputs),
				BlockIndex: int(block.Index),
			})
		}
	}

	walletAnomaly := security.AnalyzeWalletAnomalies(walletObservations)

	mempoolObservations := make([]security.MempoolTxObservation, 0)
	pendingTxs, pendingHeight := bc.PendingTransactionsWithHeightSnapshotV1()
	for _, tx := range pendingTxs {
		if tx == nil {
			continue
		}

		txid := ""
		if len(tx.ID) > 0 {
			txid = fmt.Sprintf("%x", tx.ID)
		}

		mempoolObservations = append(mempoolObservations, security.MempoolTxObservation{
			TxID:     txid,
			Sender:   tx.Sender,
			Amount:   int(bc.TransactionAmountQCAtHeightV1(tx.Amount, pendingHeight)),
			AmountQC: bc.TransactionAmountQCAtHeightV1(tx.Amount, pendingHeight),
			Inputs:   len(tx.Inputs),
			Outputs:  len(tx.Outputs),
		})
	}

	mempoolAbuse := security.AnalyzeMempoolAbuse(mempoolObservations)

	antiSpamFeePolicy := security.EvaluateDynamicAntiSpamFeePolicy(security.DynamicAntiSpamFeePolicyInput{
		Observations:     mempoolObservations,
		BaseMinFee:       0,
		MempoolSoftLimit: 25,
		MempoolHardLimit: 50,
	})

	chainHeight := 0
	blockCount := 0
	latestBlockHash := ""
	latestBlockAgeSeconds := int64(0)

	if bc != nil && bc.BlockCountSnapshot() > 0 {
		blockCount = bc.BlockCountSnapshot()
		latest := bc.TipSnapshot()
		if latest != nil {
			chainHeight = latest.Index
			if len(latest.Hash) > 0 {
				latestBlockHash = fmt.Sprintf("%x", latest.Hash)
			}
			if latest.Timestamp > 0 {
				latestBlockAgeSeconds = time.Now().Unix() - latest.Timestamp
				if latestBlockAgeSeconds < 0 {
					latestBlockAgeSeconds = 0
				}
			}
		}
	}

	finalityCheckpoint := security.EvaluateFinalityCheckpointWarning(security.FinalityCheckpointInput{
		ChainHeight:           chainHeight,
		BlockCount:            blockCount,
		LatestBlockHash:       latestBlockHash,
		LatestBlockAgeSeconds: latestBlockAgeSeconds,
		PeerCount:             0,
		FinalityDepth:         6,
		ReorgWarningDepth:     3,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"version":            "SECURITY_DASHBOARD_API_V1",
		"summary":            security.Summary("."),
		"score":              security.Score("."),
		"peerReputation":     peerReputation,
		"walletAnomaly":      walletAnomaly,
		"mempoolAbuse":       mempoolAbuse,
		"antiSpamFeePolicy":  antiSpamFeePolicy,
		"finalityCheckpoint": finalityCheckpoint,
		"endpoints": []string{
			"/api/security/summary",
			"/api/security/events",
			"/api/security/score",
			"/api/security/node-health",
			"/api/security/peer-reputation",
			"/api/security/wallet-anomaly",
			"/api/security/mempool-abuse",
			"/api/security/anti-spam-fee-policy",
			"/api/security/finality-checkpoint",
			"/api/security/dashboard",
		},
		"safeBoundary": map[string]bool{
			"readOnly":           true,
			"changesChain":       false,
			"changesBalances":    false,
			"changesConsensus":   false,
			"changesTotalSupply": false,
			"performsTransfer":   false,
			"startsMining":       false,
		},
	})
}

// WALLET_ANOMALY_API_V1
// GET /api/security/wallet-anomaly
func handleWalletAnomaly(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	observations := make([]security.WalletTxObservation, 0)

	for _, block := range bc.AllBlocksSnapshot() {
		if block == nil {
			continue
		}

		for _, tx := range block.Transactions {
			if tx == nil {
				continue
			}

			txid := ""
			if len(tx.ID) > 0 {
				txid = fmt.Sprintf("%x", tx.ID)
			}

			observations = append(observations, security.WalletTxObservation{
				TxID:       txid,
				Sender:     tx.Sender,
				Amount:     int(bc.TransactionAmountQCAtHeightV1(tx.Amount, block.Index)),
				AmountQC:   bc.TransactionAmountQCAtHeightV1(tx.Amount, block.Index),
				Inputs:     len(tx.Inputs),
				Outputs:    len(tx.Outputs),
				BlockIndex: int(block.Index),
			})
		}
	}

	writeJSON(w, http.StatusOK, security.AnalyzeWalletAnomalies(observations))
}

// MEMPOOL_ABUSE_API_V1
// GET /api/security/mempool-abuse
func handleMempoolAbuse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	observations := make([]security.MempoolTxObservation, 0)

	pendingTxs, pendingHeight := bc.PendingTransactionsWithHeightSnapshotV1()
	for _, tx := range pendingTxs {
		if tx == nil {
			continue
		}

		txid := ""
		if len(tx.ID) > 0 {
			txid = fmt.Sprintf("%x", tx.ID)
		}

		observations = append(observations, security.MempoolTxObservation{
			TxID:     txid,
			Sender:   tx.Sender,
			Amount:   int(bc.TransactionAmountQCAtHeightV1(tx.Amount, pendingHeight)),
			AmountQC: bc.TransactionAmountQCAtHeightV1(tx.Amount, pendingHeight),
			Inputs:   len(tx.Inputs),
			Outputs:  len(tx.Outputs),
		})
	}

	writeJSON(w, http.StatusOK, security.AnalyzeMempoolAbuse(observations))
}

// DYNAMIC_ANTI_SPAM_FEE_POLICY_API_V1
// GET /api/security/anti-spam-fee-policy
func handleAntiSpamFeePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	observations := make([]security.MempoolTxObservation, 0)

	pendingTxs, pendingHeight := bc.PendingTransactionsWithHeightSnapshotV1()
	for _, tx := range pendingTxs {
		if tx == nil {
			continue
		}

		txid := ""
		if len(tx.ID) > 0 {
			txid = fmt.Sprintf("%x", tx.ID)
		}

		observations = append(observations, security.MempoolTxObservation{
			TxID:     txid,
			Sender:   tx.Sender,
			Amount:   int(bc.TransactionAmountQCAtHeightV1(tx.Amount, pendingHeight)),
			AmountQC: bc.TransactionAmountQCAtHeightV1(tx.Amount, pendingHeight),
			Inputs:   len(tx.Inputs),
			Outputs:  len(tx.Outputs),
		})
	}

	result := security.EvaluateDynamicAntiSpamFeePolicy(security.DynamicAntiSpamFeePolicyInput{
		Observations:     observations,
		BaseMinFee:       0,
		MempoolSoftLimit: 25,
		MempoolHardLimit: 50,
	})

	writeJSON(w, http.StatusOK, result)
}

// FINALITY_CHECKPOINT_WARNING_API_V1
// GET /api/security/finality-checkpoint
func handleFinalityCheckpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	chainHeight := 0
	blockCount := 0
	latestBlockHash := ""
	latestBlockAgeSeconds := int64(0)

	if bc != nil && bc.BlockCountSnapshot() > 0 {
		blockCount = bc.BlockCountSnapshot()

		latest := bc.TipSnapshot()
		if latest != nil {
			chainHeight = latest.Index

			if len(latest.Hash) > 0 {
				latestBlockHash = fmt.Sprintf("%x", latest.Hash)
			}

			if latest.Timestamp > 0 {
				latestBlockAgeSeconds = time.Now().Unix() - latest.Timestamp
				if latestBlockAgeSeconds < 0 {
					latestBlockAgeSeconds = 0
				}
			}
		}
	}

	result := security.EvaluateFinalityCheckpointWarning(security.FinalityCheckpointInput{
		ChainHeight:           chainHeight,
		BlockCount:            blockCount,
		LatestBlockHash:       latestBlockHash,
		LatestBlockAgeSeconds: latestBlockAgeSeconds,
		PeerCount:             0,
		FinalityDepth:         6,
		ReorgWarningDepth:     3,
	})

	writeJSON(w, http.StatusOK, result)
}

// SECURITY_ADMIN_REPORT_API_V1
// GET /api/security/admin-report
func handleSecurityAdminReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	securitySummary := security.Summary(".")
	securityScore := security.Score(".")

	peerReputation := map[string]any{
		"version": "PEER_REPUTATION_ENGINE_V1",
		"peers":   map[string]any{},
		"status":  "NO_PEER_SECURITY",
	}
	if nodeRuntime.PeerSecurityAvailable() {
		peerReputation = nodeRuntime.PeerReputationSnapshot()
	}

	chainHeight := 0
	blockCount := 0
	latestBlockHash := ""
	latestBlockAgeSeconds := int64(0)

	if bc != nil && bc.BlockCountSnapshot() > 0 {
		blockCount = bc.BlockCountSnapshot()
		latest := bc.TipSnapshot()
		if latest != nil {
			chainHeight = latest.Index
			if len(latest.Hash) > 0 {
				latestBlockHash = fmt.Sprintf("%x", latest.Hash)
			}
			if latest.Timestamp > 0 {
				latestBlockAgeSeconds = time.Now().Unix() - latest.Timestamp
				if latestBlockAgeSeconds < 0 {
					latestBlockAgeSeconds = 0
				}
			}
		}
	}

	nodeHealth := map[string]any{
		"version":             "NODE_HEALTH_SUMMARY_V1",
		"height":              chainHeight,
		"blockCount":          blockCount,
		"peerCount":           0,
		"lastBlockAgeSeconds": latestBlockAgeSeconds,
		"health":              "GOOD",
		"syncStatus":          "LOCAL_OK",
		"syncVersion":         "NODE_SYNC_DRIFT_DETECTOR_V1",
		"safeBoundary": map[string]bool{
			"changesBalances":    false,
			"changesChain":       false,
			"changesConsensus":   false,
			"changesTotalSupply": false,
		},
	}

	if blockCount == 0 {
		nodeHealth["health"] = "WATCH"
		nodeHealth["syncStatus"] = "NO_CHAIN_DATA"
	} else if latestBlockAgeSeconds > 21600 {
		nodeHealth["health"] = "WATCH"
		nodeHealth["syncStatus"] = "NODE_STALE_BLOCKS"
	} else {
		nodeHealth["health"] = "ISOLATED"
		nodeHealth["syncStatus"] = "NO_PEERS_LOCAL_ONLY"
	}

	walletObservations := make([]security.WalletTxObservation, 0)
	if bc != nil {
		for _, block := range bc.AllBlocksSnapshot() {
			if block == nil {
				continue
			}
			for _, tx := range block.Transactions {
				if tx == nil {
					continue
				}

				txid := ""
				if len(tx.ID) > 0 {
					txid = fmt.Sprintf("%x", tx.ID)
				}

				walletObservations = append(walletObservations, security.WalletTxObservation{
					TxID:       txid,
					Sender:     tx.Sender,
					Amount:     int(bc.TransactionAmountQCAtHeightV1(tx.Amount, block.Index)),
					AmountQC:   bc.TransactionAmountQCAtHeightV1(tx.Amount, block.Index),
					Inputs:     len(tx.Inputs),
					Outputs:    len(tx.Outputs),
					BlockIndex: int(block.Index),
				})
			}
		}
	}
	walletAnomaly := security.AnalyzeWalletAnomalies(walletObservations)

	mempoolObservations := make([]security.MempoolTxObservation, 0)
	if bc != nil {
		pendingTxs, pendingHeight := bc.PendingTransactionsWithHeightSnapshotV1()
		for _, tx := range pendingTxs {
			if tx == nil {
				continue
			}

			txid := ""
			if len(tx.ID) > 0 {
				txid = fmt.Sprintf("%x", tx.ID)
			}

			mempoolObservations = append(mempoolObservations, security.MempoolTxObservation{
				TxID:     txid,
				Sender:   tx.Sender,
				Amount:   int(bc.TransactionAmountQCAtHeightV1(tx.Amount, pendingHeight)),
				AmountQC: bc.TransactionAmountQCAtHeightV1(tx.Amount, pendingHeight),
				Inputs:   len(tx.Inputs),
				Outputs:  len(tx.Outputs),
			})
		}
	}

	mempoolAbuse := security.AnalyzeMempoolAbuse(mempoolObservations)

	antiSpamFeePolicy := security.EvaluateDynamicAntiSpamFeePolicy(security.DynamicAntiSpamFeePolicyInput{
		Observations:     mempoolObservations,
		BaseMinFee:       0,
		MempoolSoftLimit: 25,
		MempoolHardLimit: 50,
	})

	finalityCheckpoint := security.EvaluateFinalityCheckpointWarning(security.FinalityCheckpointInput{
		ChainHeight:           chainHeight,
		BlockCount:            blockCount,
		LatestBlockHash:       latestBlockHash,
		LatestBlockAgeSeconds: latestBlockAgeSeconds,
		PeerCount:             0,
		FinalityDepth:         6,
		ReorgWarningDepth:     3,
	})

	report := security.BuildSecurityAdminReadonlyReport(security.SecurityAdminReportInput{
		Summary:            securitySummary,
		Score:              securityScore,
		NodeHealth:         nodeHealth,
		PeerReputation:     peerReputation,
		WalletAnomaly:      walletAnomaly,
		MempoolAbuse:       mempoolAbuse,
		AntiSpamFeePolicy:  antiSpamFeePolicy,
		FinalityCheckpoint: finalityCheckpoint,
	})

	writeJSON(w, http.StatusOK, report)
}

// AI_SECURITY_DASHBOARD_API_V1
// GET /api/ai/security-dashboard
func handleAISecurityDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	securitySummary := security.Summary(".")
	securityScore := security.Score(".")

	peerReputation := map[string]any{
		"version": "PEER_REPUTATION_ENGINE_V1",
		"peers":   map[string]any{},
		"status":  "NO_PEER_SECURITY",
	}
	if nodeRuntime.PeerSecurityAvailable() {
		peerReputation = nodeRuntime.PeerReputationSnapshot()
	}

	chainHeight := 0
	blockCount := 0
	latestBlockHash := ""
	latestBlockAgeSeconds := int64(0)

	if bc != nil && bc.BlockCountSnapshot() > 0 {
		blockCount = bc.BlockCountSnapshot()
		latest := bc.TipSnapshot()
		if latest != nil {
			chainHeight = latest.Index
			if len(latest.Hash) > 0 {
				latestBlockHash = fmt.Sprintf("%x", latest.Hash)
			}
			if latest.Timestamp > 0 {
				latestBlockAgeSeconds = time.Now().Unix() - latest.Timestamp
				if latestBlockAgeSeconds < 0 {
					latestBlockAgeSeconds = 0
				}
			}
		}
	}

	nodeHealth := map[string]any{
		"version":             "NODE_HEALTH_SUMMARY_V1",
		"height":              chainHeight,
		"blockCount":          blockCount,
		"peerCount":           0,
		"lastBlockAgeSeconds": latestBlockAgeSeconds,
		"health":              "GOOD",
		"syncStatus":          "LOCAL_OK",
		"syncVersion":         "NODE_SYNC_DRIFT_DETECTOR_V1",
		"safeBoundary": map[string]bool{
			"changesBalances":    false,
			"changesChain":       false,
			"changesConsensus":   false,
			"changesTotalSupply": false,
		},
	}

	if blockCount == 0 {
		nodeHealth["health"] = "WATCH"
		nodeHealth["syncStatus"] = "NO_CHAIN_DATA"
	} else if latestBlockAgeSeconds > 21600 {
		nodeHealth["health"] = "WATCH"
		nodeHealth["syncStatus"] = "NODE_STALE_BLOCKS"
	} else {
		nodeHealth["health"] = "ISOLATED"
		nodeHealth["syncStatus"] = "NO_PEERS_LOCAL_ONLY"
	}

	walletObservations := make([]security.WalletTxObservation, 0)
	if bc != nil {
		for _, block := range bc.AllBlocksSnapshot() {
			if block == nil {
				continue
			}
			for _, tx := range block.Transactions {
				if tx == nil {
					continue
				}

				txid := ""
				if len(tx.ID) > 0 {
					txid = fmt.Sprintf("%x", tx.ID)
				}

				walletObservations = append(walletObservations, security.WalletTxObservation{
					TxID:       txid,
					Sender:     tx.Sender,
					Amount:     int(bc.TransactionAmountQCAtHeightV1(tx.Amount, block.Index)),
					AmountQC:   bc.TransactionAmountQCAtHeightV1(tx.Amount, block.Index),
					Inputs:     len(tx.Inputs),
					Outputs:    len(tx.Outputs),
					BlockIndex: int(block.Index),
				})
			}
		}
	}
	walletAnomaly := security.AnalyzeWalletAnomalies(walletObservations)

	mempoolObservations := make([]security.MempoolTxObservation, 0)
	if bc != nil {
		pendingTxs, pendingHeight := bc.PendingTransactionsWithHeightSnapshotV1()
		for _, tx := range pendingTxs {
			if tx == nil {
				continue
			}

			txid := ""
			if len(tx.ID) > 0 {
				txid = fmt.Sprintf("%x", tx.ID)
			}

			mempoolObservations = append(mempoolObservations, security.MempoolTxObservation{
				TxID:     txid,
				Sender:   tx.Sender,
				Amount:   int(bc.TransactionAmountQCAtHeightV1(tx.Amount, pendingHeight)),
				AmountQC: bc.TransactionAmountQCAtHeightV1(tx.Amount, pendingHeight),
				Inputs:   len(tx.Inputs),
				Outputs:  len(tx.Outputs),
			})
		}
	}

	mempoolAbuse := security.AnalyzeMempoolAbuse(mempoolObservations)

	antiSpamFeePolicy := security.EvaluateDynamicAntiSpamFeePolicy(security.DynamicAntiSpamFeePolicyInput{
		Observations:     mempoolObservations,
		BaseMinFee:       0,
		MempoolSoftLimit: 25,
		MempoolHardLimit: 50,
	})

	finalityCheckpoint := security.EvaluateFinalityCheckpointWarning(security.FinalityCheckpointInput{
		ChainHeight:           chainHeight,
		BlockCount:            blockCount,
		LatestBlockHash:       latestBlockHash,
		LatestBlockAgeSeconds: latestBlockAgeSeconds,
		PeerCount:             0,
		FinalityDepth:         6,
		ReorgWarningDepth:     3,
	})

	dashboard := map[string]any{
		"version":            "SECURITY_DASHBOARD_API_V1",
		"summary":            securitySummary,
		"score":              securityScore,
		"nodeHealth":         nodeHealth,
		"peerReputation":     peerReputation,
		"walletAnomaly":      walletAnomaly,
		"mempoolAbuse":       mempoolAbuse,
		"antiSpamFeePolicy":  antiSpamFeePolicy,
		"finalityCheckpoint": finalityCheckpoint,
		"aiBridgeSource":     "AI_SECURITY_DASHBOARD_API_V1",
		"safeBoundary": map[string]bool{
			"readOnly":           true,
			"changesMempool":     false,
			"changesChain":       false,
			"changesBalances":    false,
			"changesConsensus":   false,
			"changesTotalSupply": false,
			"performsTransfer":   false,
			"startsMining":       false,
		},
	}

	adminReport := security.BuildSecurityAdminReadonlyReport(security.SecurityAdminReportInput{
		Summary:            securitySummary,
		Score:              securityScore,
		NodeHealth:         nodeHealth,
		PeerReputation:     peerReputation,
		WalletAnomaly:      walletAnomaly,
		MempoolAbuse:       mempoolAbuse,
		AntiSpamFeePolicy:  antiSpamFeePolicy,
		FinalityCheckpoint: finalityCheckpoint,
	})

	result := ai.BuildAISecurityDashboardBridge(ai.AISecurityDashboardBridgeInput{
		AdminReport: adminReport,
		Dashboard:   dashboard,
	})

	writeJSON(w, http.StatusOK, result)
}
