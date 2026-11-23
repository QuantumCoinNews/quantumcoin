package blockchain

import (
	"fmt"
	qconsensus "quantumcoin/consensus"
	"quantumcoin/wallet"
	"time"
)

// PrepareExternalMiningTemplate returns a detached, unsolved block. Reward,
// fee denomination, mempool selection and protocol gates use local mining rules.
func (bc *Blockchain) PrepareExternalMiningTemplate(miner string, difficulty int) (*Block, []byte, error) {
	if bc == nil {
		return nil, nil, ErrChainNotInitialized
	}
	if !wallet.ValidateAddress(miner) {
		return nil, nil, fmt.Errorf("invalid miner address")
	}
	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()
	if len(bc.Blocks) == 0 || bc.Blocks[len(bc.Blocks)-1] == nil {
		return nil, nil, ErrChainNotInitialized
	}
	prev := bc.Blocks[len(bc.Blocks)-1]
	height := prev.Index + 1
	if err := bc.validateLocalMiningProtocolV1(height); err != nil {
		return nil, nil, err
	}
	if err := bc.ValidateExpectedDifficultyV1(&Block{Index: height, Difficulty: difficulty}); err != nil {
		return nil, nil, err
	}
	now, err := bc.nextMiningTimeV1(time.Now().UTC())
	if err != nil {
		return nil, nil, err
	}
	txs, err := bc.miningTransactionsLockedV1(miner, height, now)
	if err != nil {
		return nil, nil, err
	}
	block := cloneBlockV1(&Block{Index: height, Timestamp: now.Unix(), Transactions: txs, PrevHash: prev.Hash, Miner: miner, Difficulty: difficulty, Metadata: map[string]string{}})
	header, err := bc.SerializeMiningHeader(block, 0)
	return block, header, err
}

// miningTransactionsLockedV1 selects the same pending snapshot, reward and fee
// denomination for in-process, external and NFT work. The caller owns stateMu.
func (bc *Blockchain) miningTransactionsLockedV1(miner string, height int, now time.Time) ([]*Transaction, error) {
	pending := bc.PendingTxs()
	fees, err := bc.CalculateBlockFeesV1(pending)
	if err != nil {
		return nil, fmt.Errorf("pending transaction fees: %w", err)
	}
	subsidy, total, err := bc.miningRewardUnitsForHeightV1(height, fees)
	if err != nil {
		return nil, err
	}
	if subsidy < 0 {
		return nil, fmt.Errorf("mining subsidy is negative")
	}
	cb, err := newCoinbaseTxWithRewardV1(miner, total, now)
	if err != nil {
		return nil, fmt.Errorf("coinbase tx: %w", err)
	}
	if err = bc.assignTransactionIDLockedV2(cb); err != nil {
		return nil, err
	}
	txs := append([]*Transaction{cb}, pending...)
	if err := bc.validateBlockTxs(txs); err != nil {
		return nil, err
	}
	return txs, nil
}

// SerializeMiningHeader selects the canonical preimage using bound activation
// policy. Both supported encodings end with an eight-byte big-endian nonce.
func (bc *Blockchain) SerializeMiningHeader(block *Block, nonce uint64) ([]byte, error) {
	if block == nil {
		return nil, ErrNilBlock
	}
	if bc.ProtocolAtHeightV1(block.Index) == qconsensus.ProtocolNFTV4 {
		header, err := canonicalPoWHeaderV4(block)
		if err != nil {
			return nil, err
		}
		return qconsensus.SerializePoWHeaderV4(header, nonce)
	}
	return SerializePoWHeaderUint64V1(block, nonce)
}
