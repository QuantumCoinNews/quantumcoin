package blockchain

// PendingTransactionsWithHeightSnapshotV1 returns detached pending transactions
// and their candidate height from the same state read boundary.
func (bc *Blockchain) PendingTransactionsWithHeightSnapshotV1() ([]*Transaction, int) {
	if bc == nil {
		return []*Transaction{}, 0
	}
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	txs := cloneTransactionsForState(bc.pendingTxs)
	if txs == nil {
		txs = []*Transaction{}
	}
	return txs, bc.nextSpendHeightV1()
}
