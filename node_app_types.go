package main

import (
	"encoding/json"
	"quantumcoin/ai"
	"quantumcoin/blockchain"
)

/* ---------- API tipleri ---------- */

type WalletResponse struct {
	Address string `json:"address"`
}

type BalanceResponse struct {
	BalanceAtoms   string  `json:"balanceAtoms"`
	SpendableAtoms string  `json:"spendableAtoms"`
	BalanceQc      string  `json:"balanceQc"`
	SpendableQc    string  `json:"spendableQc"`
	AtomsPerQc     string  `json:"atomsPerQc"`
	Balance        float64 `json:"balance"`
	Spendable      float64 `json:"spendable"`
	Height         int     `json:"height"`
}

type MineRequest struct {
	Address string `json:"address"`
}

type WebMineJobResp struct {
	GenesisHash   string `json:"genesis_hash"`
	PrevHash      string `json:"prev_hash"`
	Network       string `json:"network"`
	GenesisDomain string `json:"genesis_domain"`
	Protocol      string `json:"protocol"`
	JobID         string `json:"job_id"`
	Header        string `json:"header"`
	NonceOffset   int    `json:"nonce_offset"`
	NonceEncoding string `json:"nonce_encoding"`
	MaxNonce      string `json:"max_nonce"`
	Difficulty    int    `json:"difficulty"`
	Miner         string `json:"miner"`
	Height        int    `json:"height"`
	Expires       int64  `json:"expires"`
}

type WebMineSubmitReq struct {
	Address string      `json:"address"`
	JobID   string      `json:"job_id"`
	Nonce   json.Number `json:"nonce"`
}

type WebMineSubmitResp struct {
	Accepted bool   `json:"accepted"`
	Height   int    `json:"height"`
	Hash     string `json:"hash"`
	Message  string `json:"message,omitempty"`
}

// SendRequest carries public transaction data and signatures only.
type SendRequest struct {
	Transaction *blockchain.Transaction `json:"transaction"`
}

type SendResponse struct {
	Success bool   `json:"success"`
	TxID    string `json:"txid"`
	Message string `json:"message,omitempty"`
}

type BurnRequest = SendRequest

// TelemetryResponse documents the QuantumCoin component used by this module.
// /api/telemetry cevabı
type TelemetryResponse struct {
	AI            ai.TelemetrySnapshot `json:"ai"`
	AIPersistence ai.PersistenceStatus `json:"ai_persistence"`
	Height        int                  `json:"height"`
	BlockCount    int                  `json:"block_count"`
	Peers         int                  `json:"peers"`
	MempoolSize   int                  `json:"mempool_size"`
	MinerRunning  bool                 `json:"miner_running"`
	HTTPPort      string               `json:"http_port"`
	P2PPort       string               `json:"p2p_port"`
	ChainFile     string               `json:"chain_file"`
	TotalSupply   int                  `json:"total_supply"`
	CurrentReward int                  `json:"current_reward"`
	CPUCount      int                  `json:"cpu_count"`
	GoRoutines    int                  `json:"goroutines"`
	MemMB         float64              `json:"mem_mb"`
}
