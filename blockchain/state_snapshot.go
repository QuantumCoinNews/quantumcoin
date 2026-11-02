package blockchain

import "bytes"

// BlockchainStateSnapshot is a detached view of mutable blockchain state.
// Callers may inspect or mutate the returned values without aliasing the
// live Blockchain instance.
type BlockchainStateSnapshot struct {
	Blocks              []*Block
	UTXO                map[string][]TransactionOutput
	TotalSupply         int
	CoinbaseMaturity    int
	PendingTransactions []*Transaction
}

// ChainStateSummary is a small detached state view intended for status,
// telemetry and other hot read paths.
type ChainStateSummary struct {
	Height      int
	BlockCount  int
	TotalSupply int
	Tip         *Block
}

func cloneTransactionsForState(
	transactions []*Transaction,
) []*Transaction {
	if transactions == nil {
		return nil
	}

	cloned := make(
		[]*Transaction,
		len(transactions),
	)

	for index := range transactions {
		cloned[index] =
			cloneTransactionV1(
				transactions[index],
			)
	}

	return cloned
}

func cloneUTXOForState(
	source map[string][]TransactionOutput,
) map[string][]TransactionOutput {
	if source == nil {
		return nil
	}

	cloned := make(
		map[string][]TransactionOutput,
		len(source),
	)

	for txID, outputs := range source {
		copiedOutputs := make(
			[]TransactionOutput,
			len(outputs),
		)

		for index := range outputs {
			copiedOutputs[index] =
				cloneTransactionOutputV1(
					outputs[index],
				)
		}

		cloned[txID] = copiedOutputs
	}

	return cloned
}

// StateSnapshot returns a complete detached copy of mutable blockchain
// collections. This is intentionally not the preferred hot-path API for
// large chains; bounded snapshot methods should be used where possible.
func (bc *Blockchain) StateSnapshot() BlockchainStateSnapshot {
	if bc == nil {
		return BlockchainStateSnapshot{}
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	return BlockchainStateSnapshot{
		Blocks: cloneBlocksV1(
			bc.Blocks,
		),
		UTXO: cloneUTXOForState(
			bc.UTXO,
		),
		TotalSupply:      bc.TotalSupply,
		CoinbaseMaturity: bc.coinbaseMaturity,
		PendingTransactions: cloneTransactionsForState(
			bc.pendingTxs,
		),
	}
}

// ChainSummarySnapshot returns only the fields needed by common status and
// telemetry paths without cloning the complete chain.
func (bc *Blockchain) ChainSummarySnapshot() ChainStateSummary {
	if bc == nil {
		return ChainStateSummary{
			Height: -1,
		}
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	summary := ChainStateSummary{
		Height:      -1,
		BlockCount:  len(bc.Blocks),
		TotalSupply: bc.TotalSupply,
	}

	if len(bc.Blocks) == 0 {
		return summary
	}

	tip := bc.Blocks[len(bc.Blocks)-1]

	if tip == nil {
		return summary
	}

	summary.Height = tip.Index
	summary.Tip = cloneBlockV1(tip)

	return summary
}

// BlocksRangeSnapshot returns a bounded detached block window in canonical
// ascending slice order. total is the live block count observed under the
// same read lock.
func (bc *Blockchain) BlocksRangeSnapshot(
	start int,
	limit int,
) ([]*Block, int) {
	if bc == nil {
		return []*Block{}, 0
	}

	if limit <= 0 {
		return []*Block{}, 0
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	total := len(bc.Blocks)

	if start < 0 {
		start = 0
	}

	if start > total {
		start = total
	}

	end := start + limit

	if end < start || end > total {
		end = total
	}

	result := make(
		[]*Block,
		end-start,
	)

	for index := start; index < end; index++ {
		result[index-start] =
			cloneBlockV1(
				bc.Blocks[index],
			)
	}

	return result, total
}

// PendingTransactionsSnapshot returns detached transactions instead of a
// shallow copy of the live pointer slice.
func (bc *Blockchain) PendingTransactionsSnapshot() []*Transaction {
	if bc == nil {
		return []*Transaction{}
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	result := cloneTransactionsForState(
		bc.pendingTxs,
	)

	if result == nil {
		return []*Transaction{}
	}

	return result
}

// PendingTransactionsRangeSnapshot returns a bounded detached mempool view.
// It intentionally performs no signature verification: acceptance-time
// validation remains separate from read-time presentation.
func (bc *Blockchain) PendingTransactionsRangeSnapshot(
	start int,
	limit int,
) ([]*Transaction, int) {
	if bc == nil {
		return []*Transaction{}, 0
	}

	if limit <= 0 {
		return []*Transaction{}, 0
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	total := len(bc.pendingTxs)

	if start < 0 {
		start = 0
	}

	if start > total {
		start = total
	}

	end := start + limit

	if end < start || end > total {
		end = total
	}

	result := make(
		[]*Transaction,
		end-start,
	)

	for index := start; index < end; index++ {
		result[index-start] =
			cloneTransactionV1(
				bc.pendingTxs[index],
			)
	}

	return result, total
}

// TipSnapshot returns a detached copy of the current chain tip.
func (bc *Blockchain) TipSnapshot() *Block {
	if bc == nil {
		return nil
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	if len(bc.Blocks) == 0 {
		return nil
	}

	return cloneBlockV1(
		bc.Blocks[len(bc.Blocks)-1],
	)
}

// BlockCountSnapshot returns the block count observed under the canonical
// blockchain read lock.
func (bc *Blockchain) BlockCountSnapshot() int {
	if bc == nil {
		return 0
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	return len(bc.Blocks)
}

// TotalSupplySnapshot returns the configured total supply under the same
// synchronization boundary as other mutable Blockchain state.
func (bc *Blockchain) TotalSupplySnapshot() int {
	if bc == nil {
		return 0
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	return bc.TotalSupply
}

// BestHeightSnapshot returns the canonical tip height without exposing the
// live tip pointer.
func (bc *Blockchain) BestHeightSnapshot() int {
	if bc == nil {
		return -1
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	if len(bc.Blocks) == 0 {
		return -1
	}

	tip := bc.Blocks[len(bc.Blocks)-1]

	if tip == nil {
		return -1
	}

	return tip.Index
}

// AllBlocksSnapshot returns a complete detached block view. Hot endpoints
// should prefer BlocksRangeSnapshot so read cost remains bounded.
func (bc *Blockchain) AllBlocksSnapshot() []*Block {
	if bc == nil {
		return []*Block{}
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	result := cloneBlocksV1(
		bc.Blocks,
	)

	if result == nil {
		return []*Block{}
	}

	return result
}

// BlockByIndexSnapshot returns a detached block matching a canonical index.
func (bc *Blockchain) BlockByIndexSnapshot(
	index int,
) *Block {
	if bc == nil {
		return nil
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	for _, block := range bc.Blocks {
		if block == nil {
			continue
		}

		if block.Index == index {
			return cloneBlockV1(
				block,
			)
		}
	}

	return nil
}

// BlockByHashSnapshot returns a detached block matching hash.
func (bc *Blockchain) BlockByHashSnapshot(
	hash []byte,
) *Block {
	if bc == nil {
		return nil
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	for _, block := range bc.Blocks {
		if block == nil {
			continue
		}

		if bytes.Equal(
			block.Hash,
			hash,
		) {
			return cloneBlockV1(
				block,
			)
		}
	}

	return nil
}

// BalanceSnapshot executes the legacy balance calculation while holding the
// canonical read lock. GetBalance itself remains the lock-free internal
// primitive until the remaining call graph has been migrated.
func (bc *Blockchain) BalanceSnapshot(
	address string,
) int {
	if bc == nil {
		return 0
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	spendHeight := bc.nextSpendHeightV1()

	return bc.balanceInProtocolUnitsAtHeightLockedV1(
		address,
		spendHeight,
		false,
	)
}

// SpendableBalanceSnapshot protects the complete spendable-balance read
// against concurrent chain/UTXO mutation.
func (bc *Blockchain) SpendableBalanceSnapshot(
	address string,
) int {
	if bc == nil {
		return 0
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	spendHeight := bc.nextSpendHeightV1()

	return bc.balanceInProtocolUnitsAtHeightLockedV1(
		address,
		spendHeight,
		true,
	)
}

// ExpectedSubsidyForHeightSnapshot gives API/status callers a synchronized
// reward calculation without adding a nested lock to consensus internals.
func (bc *Blockchain) ExpectedSubsidyForHeightSnapshot(
	height int,
) int {
	if bc == nil {
		return 0
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	return bc.ExpectedSubsidyForHeightV1(
		height,
	)
}

// PendingTransactionCountSnapshot returns the count of transactions in the
// canonical Blockchain-owned pending transaction pool.
func (bc *Blockchain) PendingTransactionCountSnapshot() int {
	if bc == nil {
		return 0
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	return len(bc.pendingTxs)
}
