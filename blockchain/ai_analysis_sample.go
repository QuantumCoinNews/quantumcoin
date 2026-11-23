package blockchain

import (
	"time"

	"quantumcoin/ai"
)

// AIConfirmedBlockSample is a bounded advisory projection taken after commit.
// No storage or external callbacks execute while the chain lock is held.
func (bc *Blockchain) AIConfirmedBlockSample(height int) []ai.TxLite {
	if bc == nil {
		return nil
	}
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	if height < 0 || height >= len(bc.Blocks) {
		return nil
	}
	block := bc.Blocks[height]
	if block == nil || block.Index != height || block.Timestamp <= 0 {
		return nil
	}
	out := make([]ai.TxLite, 0)
	budget := analysisByteBudget
	for i, tx := range block.Transactions {
		if i >= AnalysisMaxTransactions {
			break
		}
		if !safeAnalysisTransaction(tx) {
			continue
		}
		cost := analysisTransactionCost(tx)
		if cost > budget {
			break
		}
		budget -= cost
		if sample, ok := bc.verifiedAIProjectionV1(tx, height, time.Unix(block.Timestamp, 0).UTC()); ok {
			out = append(out, sample)
		}
	}
	return out
}

// AIAnalysisSample observes the canonical pending pool and recent confirmed
// transactions under one read lock. It never admits, removes or rejects a
// transaction. Limits apply to visited entries, including malformed entries.
// Pending entries take priority; confirmed history survives mining/restarts
// through the chain itself, without a second mempool or unbounded cache.
func (bc *Blockchain) AIAnalysisSample(address string) ([]ai.TxLite, bool) {
	if bc == nil {
		return nil, false
	}
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	out := make([]ai.TxLite, 0)
	seen := make(map[string]bool)
	visited, budget := 0, analysisByteBudget
	truncated := false
	appendTx := func(tx *Transaction, height int, observedAt time.Time) bool {
		if visited >= AnalysisMaxTransactions {
			truncated = true
			return false
		}
		visited++
		if !safeAnalysisTransaction(tx) {
			truncated = true
			return true
		}
		cost := analysisTransactionCost(tx)
		if cost > budget {
			truncated = true
			return false
		}
		budget -= cost
		if tx.IsCoinbase() || (address != "" && tx.Sender != address) {
			return true
		}
		if seen[transactionLocationIndexKey(tx.ID)] {
			return true
		}
		lite, ok := bc.verifiedAIProjectionV1(tx, height, observedAt)
		if !ok {
			truncated = true
			return true
		}
		seen[lite.TxID] = true
		out = append(out, lite)
		return true
	}
	for _, tx := range bc.pendingTxs {
		var observedAt time.Time
		if tx != nil {
			observedAt = bc.pendingObservedAt[transactionLocationIndexKey(tx.ID)]
		}
		if !appendTx(tx, bc.nextSpendHeightV1(), observedAt) {
			return out, truncated
		}
	}
	start := len(bc.Blocks) - AnalysisMaxBlocks
	if start < 0 {
		start = 0
	} else if start > 0 {
		truncated = true
	}
	for i := len(bc.Blocks) - 1; i >= start; i-- {
		block := bc.Blocks[i]
		if block == nil || block.Index != i || block.Timestamp <= 0 {
			continue
		}
		for _, tx := range block.Transactions {
			if !appendTx(tx, i, time.Unix(block.Timestamp, 0).UTC()) {
				return out, truncated
			}
		}
	}
	return out, truncated
}
