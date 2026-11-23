package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"quantumcoin/blockchain"
	"quantumcoin/config"
	qnode "quantumcoin/node"
	"quantumcoin/p2p"
	"quantumcoin/wallet"
	"strconv"
	"strings"
	"time"
)

/* ---------- Mining döngüleri ---------- */

func startContinuousMining(miner string) {
	if err := beginContinuousMining(miner); err != nil {
		log.Printf("miner start error: %v", err)
		return
	}
	nodeRuntime.WaitMining()
}

func beginContinuousMining(miner string) error {
	if nodeRuntime == nil {
		return fmt.Errorf("node runtime unavailable")
	}
	if !wallet.ValidateAddress(strings.TrimSpace(miner)) {
		return fmt.Errorf("valid wallet address required")
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
	return err
}

func handleMineBlock(w http.ResponseWriter, r *http.Request) {
	mineBlockHTTPV1(w, r, false)
}

func handleIsolatedLocalMineBlock(w http.ResponseWriter, r *http.Request) {
	mineBlockHTTPV1(w, r, true)
}

func mineBlockHTTPV1(w http.ResponseWriter, r *http.Request, isolatedLocal bool) {
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
	bc, _ := nodeRuntime.Snapshot()
	if bc == nil {
		writeError(w, http.StatusServiceUnavailable, "mining blockchain is unavailable")
		return
	}
	mine := nodeRuntime.MineOnceV1
	if isolatedLocal {
		mine = func(miner string, difficulty int) (*blockchain.Block, error) {
			return nodeRuntime.MineIsolatedLocalBlockV1(miner, difficulty, r.Header.Get("X-QC-Expected-Genesis"))
		}
	}
	block, err := mine(req.Address, currentEffectiveDifficultyBits())
	if err != nil {
		if errors.Is(err, qnode.ErrMiningNetworkMismatchV1) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, qnode.ErrRuntimeReadOnlyV1) {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	nodeRuntime.BroadcastP2P(p2p.BlockMessage(block))

	writeOK(w, map[string]any{
		"success":    true,
		"reward":     bc.ExpectedSubsidyForHeightSnapshot(block.Index),
		"height":     block.Index,
		"block_hash": hex.EncodeToString(block.Hash),
	})

	// processAIBonus() disabled: AI monitors only, no mining bonus / no supply effect.
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
	bc, _ := nodeRuntime.Snapshot()
	if bc.BlockCountSnapshot() > 0 {
		last := bc.TipSnapshot()
		nodeRuntime.BroadcastP2P(p2p.BlockMessage(last))
	}

	writeOK(w, map[string]any{
		"success": true,
		"mined":   n,
		"height":  bc.BestHeightSnapshot(),
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

	bc, _ := nodeRuntime.Snapshot()
	if bc != nil {
		dynamicBits = bc.NextDifficultySnapshot(
			cfg.DefaultDifficultyBits,
		)
	}

	return effectiveDifficultyBits(dynamicBits)
}

/* ---------- Miner control (start/stop/status) ---------- */

type minerStartReq struct {
	Address string `json:"address"`
}

// Only one explicit JSON address is accepted; malformed input must never
// fall back to configuration or retarget an already-running miner silently.
func readMinerControlAddressV1(w http.ResponseWriter, r *http.Request) (string, bool) {
	fail := func(err error) (string, bool) {
		if err != nil && writeRequestBodyTooLargeV1(w, err) {
			return "", false
		}
		writeError(w, http.StatusBadRequest, "one explicit valid JSON miner address is required")
		return "", false
	}
	if r.Body == nil {
		return fail(nil)
	}
	defer r.Body.Close()
	if r.URL.RawQuery != "" {
		return fail(nil)
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, apiMaxBodyBytes))
	opening, err := decoder.Token()
	if err != nil {
		return fail(err)
	}
	if opening != json.Delim('{') || !decoder.More() {
		return fail(nil)
	}
	key, err := decoder.Token()
	if err != nil {
		return fail(err)
	}
	if key != "address" {
		return fail(nil)
	}
	var address string
	if err = decoder.Decode(&address); err != nil {
		return fail(err)
	}
	if decoder.More() {
		return fail(nil)
	}
	closing, err := decoder.Token()
	if err != nil {
		return fail(err)
	}
	if closing != json.Delim('}') {
		return fail(nil)
	}
	if err = decoder.Decode(new(any)); err != io.EOF {
		return fail(err)
	}
	address = strings.TrimSpace(address)
	if !wallet.ValidateAddress(address) {
		return fail(nil)
	}
	return address, true
}

func handleMinerStart(w http.ResponseWriter, r *http.Request) {
	// --- Güvenlik: Miner'ı uzaktan başlatmak için token iste ---
	if !checkAPIToken(w, r) {
		return
	}

	if !requireDevMinerEndpointAllowedV1(w, r, "/api/miner/start") {
		return
	}
	addr, ok := readMinerControlAddressV1(w, r)
	if !ok {
		return
	}

	bc, _ := nodeRuntime.Snapshot()
	if bc == nil {
		writeError(w, http.StatusServiceUnavailable, "node runtime unavailable")
		return
	}
	if nodeRuntime != nil && nodeRuntime.IsMining() {
		if nodeRuntime.MinerAddress() != addr {
			writeError(w, http.StatusConflict, "miner already running for another address")
			return
		}
		writeOK(w, map[string]any{
			"running":    true,
			"message":    "miner already running",
			"address":    nodeRuntime.MinerAddress(),
			"difficulty": currentEffectiveDifficultyBits(),
			"height":     bc.BestHeightSnapshot(),
		})
		return
	}

	if err := beginContinuousMining(addr); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

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

	bc, _ := nodeRuntime.Snapshot()
	if bc == nil {
		writeError(w, http.StatusServiceUnavailable, "node runtime unavailable")
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
		"running":  true,
		"stopping": true,
		"message":  "miner stop requested",
		"height":   bc.BestHeightSnapshot(),
	})
}

func handleMinerStatus(w http.ResponseWriter, r *http.Request) {
	if !requireDevMinerEndpointAllowedV1(w, r, "/api/miner/status") {
		return
	}
	bc, _ := nodeRuntime.Snapshot()
	if bc == nil {
		writeError(w, http.StatusServiceUnavailable, "node runtime unavailable")
		return
	}
	writeOK(w, map[string]any{
		"running":  nodeRuntime != nil && nodeRuntime.IsMining(),
		"height":   bc.BestHeightSnapshot(),
		"bits":     currentEffectiveDifficultyBits(),
		"httpPort": getHTTPPort(),
	})
}
