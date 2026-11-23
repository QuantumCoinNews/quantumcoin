package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"quantumcoin/blockchain"
	qnode "quantumcoin/node"
	"quantumcoin/p2p"
	"strconv"
	"strings"
	"sync"
	"time"
)

const webMiningProtocol = "QC_WEB_BLOCK_POW_V1"
const maxWebMiningJobs = 256
const maxWebMiningJobsPerMiner = 8

type webJob struct {
	Template  *blockchain.Block
	Owner     *qnode.Runtime `json:"-"`
	Miner     string
	ExpiresAt time.Time
}

var (
	jobMu   sync.Mutex
	webJobs = make(map[string]*webJob)
)

func handleMineJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, 405, "method not allowed")
		return
	}
	addr := strings.TrimSpace(r.URL.Query().Get("address"))
	state := nodeRuntime.MiningTip()
	jobMu.Lock()
	defer jobMu.Unlock()
	count := 0
	for id, j := range webJobs {
		if time.Now().After(j.ExpiresAt) || j.Owner != nodeRuntime || j.Template == nil || hex.EncodeToString(j.Template.PrevHash) != state.Hash {
			delete(webJobs, id)
			continue
		}
		if j.Miner == addr {
			count++
		}
	}
	if len(webJobs) >= maxWebMiningJobs || count >= maxWebMiningJobsPerMiner {
		writeError(w, 429, "mining job capacity reached; wait for expiry")
		return
	}
	template, header, err := nodeRuntime.PrepareExternalMining(addr)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		writeError(w, 500, "cannot issue job ID")
		return
	}
	id := hex.EncodeToString(token[:])
	expires := time.Now().Add(time.Minute)
	webJobs[id] = &webJob{Template: template, Owner: nodeRuntime, Miner: addr, ExpiresAt: expires}
	state = nodeRuntime.MiningTip()
	if state.Hash != hex.EncodeToString(template.PrevHash) {
		delete(webJobs, id)
		writeError(w, 409, "mining tip changed; request current work")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeOK(w, WebMineJobResp{GenesisHash: state.GenesisHash, PrevHash: hex.EncodeToString(template.PrevHash), Network: state.Network, GenesisDomain: state.GenesisDomain, Protocol: webMiningProtocol, JobID: id, Header: hex.EncodeToString(header), NonceOffset: len(header) - 8, NonceEncoding: "uint64-big-endian", MaxNonce: strconv.FormatUint(blockchain.MaxConsensusNonceV1, 10), Difficulty: template.Difficulty, Miner: addr, Height: template.Index, Expires: expires.Unix()})
}

func handleMiningTip(w http.ResponseWriter, r *http.Request) {
	if !requireAPIGet(w, r) {
		return
	}
	state := nodeRuntime.MiningTip()
	if state.Height < 0 {
		writeError(w, 503, "mining chain unavailable")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeOK(w, state)
}

func handleMiningResult(w http.ResponseWriter, r *http.Request) {
	if !requireAPIGet(w, r) {
		return
	}
	height, err := strconv.Atoi(r.URL.Query().Get("height"))
	hash, hashErr := hex.DecodeString(r.URL.Query().Get("hash"))
	address := r.URL.Query().Get("address")
	if err != nil || height < 1 || hashErr != nil || len(hash) != 32 || address == "" {
		writeError(w, 400, "valid height, hash and reward address required")
		return
	}
	chain, _ := nodeRuntime.Snapshot()
	state := chain.MiningTipSnapshot()
	w.Header().Set("Cache-Control", "no-store")
	writeOK(w, map[string]any{"accepted": chain.MiningResultSnapshot(height, hash, address), "hash": hex.EncodeToString(hash), "height": height, "miner": address, "genesis_hash": state.GenesisHash, "rewardStatus": "coinbase-subject-to-maturity-and-reorg"})
}

func handleMineSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "method not allowed")
		return
	}
	defer r.Body.Close()
	var req WebMineSubmitReq
	body, err := io.ReadAll(r.Body)
	if err != nil {
		if !writeRequestBodyTooLargeV1(w, err) {
			writeError(w, 400, "invalid json")
		}
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, 400, "invalid json")
		return
	}
	nonce, err := strconv.ParseUint(string(req.Nonce), 10, 64)
	if err != nil || nonce > blockchain.MaxConsensusNonceV1 || req.JobID == "" || req.Address == "" {
		writeError(w, 400, "job_id, address and valid decimal nonce required")
		return
	}
	// Admission, durable commit and consumption share one lock. A duplicate
	// cannot pass admission while the first request is still committing.
	jobMu.Lock()
	j := webJobs[req.JobID]
	if j == nil || j.Owner != nodeRuntime || time.Now().After(j.ExpiresAt) {
		delete(webJobs, req.JobID)
		jobMu.Unlock()
		writeError(w, 409, "unknown, consumed or expired job")
		return
	}
	if req.Address != j.Miner {
		jobMu.Unlock()
		writeError(w, 400, "address mismatch")
		return
	}
	block, err := j.Owner.CommitExternalMining(j.Template, nonce)
	if err != nil {
		// Invalid PoW and storage failures never produce an accepted receipt.
		// Retain the immutable template for a valid retry until expiry.
		jobMu.Unlock()
		writeError(w, 409, err.Error())
		return
	}
	delete(webJobs, req.JobID)
	jobMu.Unlock()
	j.Owner.BroadcastP2P(p2p.BlockMessage(block))
	writeOK(w, WebMineSubmitResp{Accepted: true, Hash: hex.EncodeToString(block.Hash), Height: block.Index, Message: "block committed locally"})
}
