package blockchain

// AnalysisMaxBlocks Bounds apply before cloning. A large chain, mempool or transaction cannot
// turn one advisory request into an unbounded allocation or full-chain scan.
const AnalysisMaxBlocks = 200
const AnalysisMaxTransactions = 2000
const analysisByteBudget = 4 << 20

func analysisTransactionCost(tx *Transaction) int {
	return 512 + len(tx.Inputs)*384 + len(tx.Outputs)*96
}

func safeAnalysisTransaction(tx *Transaction) bool {
	if tx == nil || len(tx.ID) > 32 || len(tx.Sender) > 128 || len(tx.Inputs) > 128 || len(tx.Outputs) > 128 {
		return false
	}
	for _, in := range tx.Inputs {
		if len(in.TxID) > 32 || len(in.Signature) > 80 || len(in.PubKey) > 65 {
			return false
		}
	}
	for _, out := range tx.Outputs {
		if len(out.PubKeyHash) > 32 {
			return false
		}
	}
	return true
}

func (bc *Blockchain) AnalysisBlocksSnapshot() ([]*Block, bool) {
	if bc == nil {
		return []*Block{}, false
	}
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	start := len(bc.Blocks) - AnalysisMaxBlocks
	if start < 0 {
		start = 0
	}
	truncated := start > 0
	visited := 0
	budget := analysisByteBudget
	blocks := make([]*Block, 0, len(bc.Blocks)-start)
	for i := len(bc.Blocks) - 1; i >= start; i-- {
		source := bc.Blocks[i]
		if source == nil {
			continue
		}
		block := &Block{Index: source.Index, Transactions: []*Transaction{}}
		for _, tx := range source.Transactions {
			if visited >= AnalysisMaxTransactions {
				truncated = true
				break
			}
			visited++
			if !safeAnalysisTransaction(tx) {
				truncated = true
				continue
			}
			cost := analysisTransactionCost(tx)
			if cost > budget {
				truncated = true
				continue
			}
			budget -= cost
			detached := cloneBlockV1(&Block{Transactions: []*Transaction{tx}})
			block.Transactions = append(block.Transactions, detached.Transactions[0])
		}
		blocks = append(blocks, block)
		if visited >= AnalysisMaxTransactions {
			if i > start {
				truncated = true
			}
			break
		}
	}
	for i, j := 0, len(blocks)-1; i < j; i, j = i+1, j-1 {
		blocks[i], blocks[j] = blocks[j], blocks[i]
	}
	return blocks, truncated
}

func (bc *Blockchain) AnalysisPendingSnapshot() ([]*Transaction, int, bool) {
	if bc == nil {
		return []*Transaction{}, 0, false
	}
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	count := len(bc.pendingTxs)
	truncated := count > AnalysisMaxTransactions
	if count > AnalysisMaxTransactions {
		count = AnalysisMaxTransactions
	}
	txs := make([]*Transaction, 0, count)
	budget := analysisByteBudget
	for _, tx := range bc.pendingTxs[:count] {
		if !safeAnalysisTransaction(tx) {
			truncated = true
			continue
		}
		cost := analysisTransactionCost(tx)
		if cost > budget {
			truncated = true
			continue
		}
		budget -= cost
		copy := cloneBlockV1(&Block{Transactions: []*Transaction{tx}})
		txs = append(txs, copy.Transactions[0])
	}
	return txs, bc.nextSpendHeightV1(), truncated
}

type BlockPageSummary struct {
	Index               int
	Hash, PrevHash      []byte
	Timestamp           int64
	Miner               string
	Difficulty, TxCount int
}

func (bc *Blockchain) BlockSummaryPageFromTipSnapshot(offset, limit int) ([]BlockPageSummary, int) {
	if bc == nil {
		return []BlockPageSummary{}, 0
	}
	if offset < 0 {
		offset = 0
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	total := len(bc.Blocks)
	if offset > total {
		offset = total
	}
	end := total - offset
	start := end - limit
	if start < 0 {
		start = 0
	}
	result := make([]BlockPageSummary, 0, end-start)
	for _, b := range bc.Blocks[start:end] {
		if b != nil {
			result = append(result, BlockPageSummary{Index: b.Index, Hash: append([]byte(nil), b.Hash...), PrevHash: append([]byte(nil), b.PrevHash...), Timestamp: b.Timestamp, Miner: b.Miner, Difficulty: b.Difficulty, TxCount: len(b.Transactions)})
		}
	}
	return result, total
}
