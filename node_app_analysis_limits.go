package main

import (
	"net/http"
	"quantumcoin/blockchain"
	"strconv"
)

var analysisSlotsV1 = make(chan struct{}, 2)

func withAnalysisBudgetV1(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case analysisSlotsV1 <- struct{}{}:
			defer func() { <-analysisSlotsV1 }()
			next(w, r)
		default:
			writeError(w, http.StatusTooManyRequests, "analysis concurrency limit reached")
		}
	}
}

func analysisHeadersV1(w http.ResponseWriter, truncated bool) {
	w.Header().Set("X-Analysis-Scope", "bounded-recent-sample")
	w.Header().Set("X-Analysis-Max-Blocks", strconv.Itoa(blockchain.AnalysisMaxBlocks))
	w.Header().Set("X-Analysis-Max-Transactions", strconv.Itoa(blockchain.AnalysisMaxTransactions))
	if truncated {
		w.Header().Set("X-Analysis-Truncated", "true")
	} else if w.Header().Get("X-Analysis-Truncated") == "" {
		w.Header().Set("X-Analysis-Truncated", "false")
	}
}
func boundedAnalysisBlocksV1(w http.ResponseWriter, bc *blockchain.Blockchain) []*blockchain.Block {
	blocks, truncated := bc.AnalysisBlocksSnapshot()
	analysisHeadersV1(w, truncated)
	return blocks
}
func boundedAnalysisPendingV1(w http.ResponseWriter, bc *blockchain.Blockchain) ([]*blockchain.Transaction, int) {
	txs, height, truncated := bc.AnalysisPendingSnapshot()
	analysisHeadersV1(w, truncated)
	return txs, height
}
