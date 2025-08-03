package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"quantumcoin/ai"
	"quantumcoin/blockchain"
	"quantumcoin/game"
	"quantumcoin/internal"
	"quantumcoin/p2p"
	"quantumcoin/wallet"
	"strconv"
	"strings"
	"time"
)

const blockchainFile = "chain_data.dat"

// --- API Veri Tipleri ---
type WalletResponse struct {
	Address string `json:"address"`
}
type BalanceResponse struct {
	Balance float64 `json:"balance"`
}
type MineRequest struct {
	Address string `json:"address"`
}

// --- GLOBAL ---
var (
	bc        *blockchain.Blockchain
	gameState = game.NewGameState() // Oyun state
)

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  run [port]               - Run node on port")
	fmt.Println("  connect [port] [address] - Connect to peer")
	fmt.Println("  send [from] [to] [amt]   - Send coins")
	fmt.Println("  mine [miner]             - Mine a new block")
	fmt.Println("  print                    - Print blockchain")
	fmt.Println("  newaddr                  - Generate a new wallet address")
	fmt.Println("  api                      - Start HTTP API (default: 8081)")
}

func main() {
	var err error

	if _, err = os.Stat(blockchainFile); err == nil {
		bc, err = blockchain.LoadBlockchainFromFile(blockchainFile)
		if err != nil {
			log.Fatalf("Blockchain yüklenemedi: %v", err)
		}
	} else {
		bc = blockchain.NewBlockchain(50, 25500000)
	}

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "run":
		if len(os.Args) < 3 {
			fmt.Println("Port number missing")
			return
		}
		port := os.Args[2]
		go startHTTPAPI()
		p2p.RunNode(port, bc)
	case "api":
		startHTTPAPI()
	case "connect":
		if len(os.Args) < 4 {
			fmt.Println("Usage: connect [port] [address]")
			return
		}
		port := os.Args[2]
		address := os.Args[3]
		go startHTTPAPI()
		p2p.ConnectToPeer(port, address, bc)
	case "send":
		if len(os.Args) < 5 {
			fmt.Println("Usage: send [from] [to] [amount]")
			return
		}
		from := os.Args[2]
		to := os.Args[3]
		amount, err := strconv.Atoi(os.Args[4])
		if err != nil {
			fmt.Println("Invalid amount")
			return
		}
		tx, err := blockchain.NewTransaction(from, to, amount, bc)
		if err != nil {
			log.Println("Transaction creation failed:", err)
			return
		}
		err = bc.AddTransaction(tx)
		if err != nil {
			log.Println("Transaction failed:", err)
		} else {
			fmt.Println("Transaction added to pool")
		}
	case "mine":
		if len(os.Args) < 3 {
			fmt.Println("Usage: mine [miner]")
			return
		}
		miner := os.Args[2]
		block, err := bc.MineBlock(miner, 16)
		if err != nil {
			log.Println("Mining failed:", err)
		} else {
			fmt.Printf("✅ New block mined by %s with hash %x\n", miner, block.Hash)
		}
		processAIBonus()
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

	// Zinciri kaydet
	err = bc.SaveToFile(blockchainFile)
	if err != nil {
		log.Fatalf("Blockchain kaydedilemedi: %v", err)
	}
}

// --- HTTP API ---
func startHTTPAPI() {
	http.HandleFunc("/api/wallet/new", handleNewWallet)
	http.HandleFunc("/api/wallet/balance/", handleBalance)
	http.HandleFunc("/api/mine", handleMineBlock)
	http.HandleFunc("/api/ai/bonus", handleAIBonus)
	http.HandleFunc("/api/ai/analysis", handleAIAnalysis)
	http.HandleFunc("/api/game/score", handleGameScore)
	http.HandleFunc("/api/game/leaderboard", handleLeaderboard)
	fmt.Println("HTTP API started at http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func handleNewWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" && r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	wal := wallet.NewWallet()
	res := WalletResponse{Address: wal.GetAddress()}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func handleBalance(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 5 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	address := parts[4]
	balance := bc.GetBalance(address)
	res := BalanceResponse{Balance: float64(balance)}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func handleMineBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	body, _ := io.ReadAll(r.Body)
	var req MineRequest
	json.Unmarshal(body, &req)
	if req.Address == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false, "message":"address is required"}`))
		return
	}
	block, err := bc.MineBlock(req.Address, 16)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf(`{"success":false, "message":"%s"}`, err.Error())))
		return
	}
	res := map[string]interface{}{
		"success":    true,
		"reward":     50,
		"block_hash": fmt.Sprintf("%x", block.Hash),
	}
	processAIBonus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// --- YENİ: AI/Bonus Otomasyonunu Tetikleyen Fonksiyon ---
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bonuses)
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
	result := map[string]interface{}{
		"anomaly_report":     anomalies,
		"recommendations":    recs,
		"reward_suggestions": suggestions,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// --- GAME API (opsiyonel, entegre edilmiş hali) ---
func handleGameScore(w http.ResponseWriter, r *http.Request) {
	player := r.URL.Query().Get("player")
	score, _ := strconv.Atoi(r.URL.Query().Get("score"))
	game.HandleTelegramScore(gameState, player, score)
	w.Write([]byte(`{"success":true}`))
}

func handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	top := game.GetTopPlayers(gameState, 10)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(top)
}
