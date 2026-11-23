package miner

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"quantumcoin/apiclient"
	"quantumcoin/blockchain"
)

var errNodeMiningDenied = errors.New("node denied mining access")

type nodeBackend struct {
	base, token string
	client      *http.Client
	mu          sync.Mutex
	genesis     string
}

// NewNodeBackend uses the running node's canonical template and commit service.
func NewNodeBackend(base, token string) (Backend, error) {
	token = strings.TrimSpace(token)
	u, err := apiclient.ValidateNodeURL(strings.TrimSpace(base))
	if err != nil {
		return nil, err
	}
	if token == "" || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.ForceQuery {
		return nil, errors.New("node mining requires an origin and QC_API_TOKEN")
	}
	return &nodeBackend{base: strings.TrimRight(u.String(), "/"), token: token, client: apiclient.NewNodeClient(15 * time.Second)}, nil
}

// NewPinnedNodeBackend Expected genesis comes from local trusted configuration, never caller work.
func NewPinnedNodeBackend(base, token, genesis string) (Backend, error) {
	backend, err := NewNodeBackend(base, token)
	if err != nil {
		return nil, err
	}
	raw, err := hex.DecodeString(genesis)
	if err != nil || len(raw) != 32 {
		return nil, errors.New("valid expected mining genesis required")
	}
	backend.(*nodeBackend).genesis = strings.ToLower(genesis)
	return backend, nil
}

func PrepareNodeWorker(ctx context.Context, base, token, genesis string, cfg WorkerConfig) (*Worker, error) {
	backend, err := NewPinnedNodeBackend(base, token, genesis)
	if err != nil {
		return nil, err
	}
	first, err := backend.GetWork(ctx, cfg.Address)
	if err != nil {
		return nil, err
	}
	worker := NewWorker(backend, cfg)
	worker.initial = first
	return worker, nil
}

func (n *nodeBackend) request(ctx context.Context, method, path string, body []byte, dst any) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, n.base+path, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("X-QC-API-KEY", n.token)
	req.Header.Set("Accept", "application/json")
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := n.client.Do(req)
	if err != nil {
		return 0, errors.New("node mining request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return resp.StatusCode, errNodeMiningDenied
	}
	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, nil
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 65537))
	if err != nil || len(raw) > 65536 || json.Unmarshal(raw, dst) != nil {
		return resp.StatusCode, errors.New("invalid node mining response")
	}
	return resp.StatusCode, nil
}

func (n *nodeBackend) GetWork(ctx context.Context, address string) (*Work, error) {
	var out struct {
		GenesisHash   string `json:"genesis_hash"`
		PrevHash      string `json:"prev_hash"`
		Expires       int64  `json:"expires"`
		Protocol      string `json:"protocol"`
		JobID         string `json:"job_id"`
		Header        string `json:"header"`
		NonceOffset   int    `json:"nonce_offset"`
		NonceEncoding string `json:"nonce_encoding"`
		MaxNonce      string `json:"max_nonce"`
		Difficulty    int    `json:"difficulty"`
		Miner         string `json:"miner"`
		Height        int    `json:"height"`
	}
	status, err := n.request(ctx, http.MethodGet, "/api/mine/job?address="+url.QueryEscape(address), nil, &out)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("node mining job HTTP %d", status)
	}
	header, headerErr := hex.DecodeString(out.Header)
	job, jobErr := hex.DecodeString(out.JobID)
	genesis, genesisErr := hex.DecodeString(out.GenesisHash)
	previous, previousErr := hex.DecodeString(out.PrevHash)
	if headerErr != nil || len(header) < 8 || jobErr != nil || len(job) != 32 ||
		genesisErr != nil || len(genesis) != 32 || previousErr != nil || len(previous) != 32 || out.Expires <= time.Now().Unix() || out.Expires > time.Now().Add(90*time.Second).Unix() ||
		out.Protocol != "QC_WEB_BLOCK_POW_V1" || out.NonceOffset != len(header)-8 ||
		out.NonceEncoding != "uint64-big-endian" || out.MaxNonce != strconv.FormatUint(blockchain.MaxConsensusNonceV1, 10) ||
		out.Miner != address || out.Height < 1 || out.Difficulty < 1 || out.Difficulty > 255 {
		return nil, errors.New("invalid node mining job")
	}
	n.mu.Lock()
	if n.genesis != "" && out.GenesisHash != n.genesis {
		n.mu.Unlock()
		return nil, errNodeMiningDenied
	}
	if out.GenesisHash != "" {
		n.genesis = out.GenesisHash
	}
	n.mu.Unlock()
	var expires time.Time
	if out.Expires > 0 {
		expires = time.Unix(out.Expires, 0)
	}
	return &Work{Left: []byte(out.JobID), Header: header, Target: targetFromBits(out.Difficulty), Height: out.Height, Miner: address, GenesisHash: out.GenesisHash, PrevHash: out.PrevHash, ExpiresAt: expires}, nil
}

func (n *nodeBackend) WorkCurrent(ctx context.Context, work *Work) (bool, error) {
	if !work.ExpiresAt.IsZero() && time.Now().After(work.ExpiresAt) {
		return false, nil
	}
	var tip blockchain.MiningTip
	status, err := n.request(ctx, http.MethodGet, "/api/mine/tip", nil, &tip)
	if err != nil {
		return false, err
	}
	if status != 200 || tip.Version != "QC_MINING_TIP_V1" {
		return false, errors.New("mining tip unavailable")
	}
	if work.GenesisHash != "" && tip.GenesisHash != work.GenesisHash {
		return false, errNodeMiningDenied
	}
	return tip.Height+1 == work.Height && (work.PrevHash == "" || tip.Hash == work.PrevHash), nil
}

func (n *nodeBackend) Submit(ctx context.Context, work *Work, nonce uint64, hash string) (bool, error) {
	if work == nil || nonce > blockchain.MaxConsensusNonceV1 {
		return false, errors.New("invalid mining solution")
	}
	body, err := json.Marshal(map[string]any{"job_id": string(work.Left), "address": work.Miner, "nonce": nonce})
	if err != nil {
		return false, err
	}
	var out struct {
		Accepted bool   `json:"accepted"`
		Hash     string `json:"hash"`
		Height   int    `json:"height"`
	}
	status, err := n.request(ctx, http.MethodPost, "/api/mine/submit", body, &out)
	if err != nil {
		if ctx.Err() == nil && !errors.Is(err, errNodeMiningDenied) {
			lookupCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			var saved struct {
				Accepted bool   `json:"accepted"`
				Hash     string `json:"hash"`
				Height   int    `json:"height"`
				Miner    string `json:"miner"`
				Genesis  string `json:"genesis_hash"`
			}
			path := "/api/mine/result?height=" + strconv.Itoa(work.Height) + "&hash=" + url.QueryEscape(hash) + "&address=" + url.QueryEscape(work.Miner)
			code, lookupErr := n.request(lookupCtx, http.MethodGet, path, nil, &saved)
			if lookupErr == nil && code == 200 && saved.Accepted && saved.Hash == hash && saved.Height == work.Height && saved.Miner == work.Miner && saved.Genesis == work.GenesisHash {
				return true, nil
			}
		}
		return false, err
	}
	if status == http.StatusConflict || status == http.StatusTooManyRequests {
		return false, nil
	}
	if status != http.StatusOK || !out.Accepted || out.Height != work.Height || !strings.EqualFold(out.Hash, hash) {
		return false, errors.New("node did not return a matching mining receipt")
	}
	return true, nil
}
