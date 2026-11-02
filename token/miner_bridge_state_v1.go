package token

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"quantumcoin/blockchain"
	"quantumcoin/miner"
)

const (
	MinerTokenBridgeStateVersionV1 = "QC_MINER_TOKEN_BRIDGE_STATE_V1"
	MinerTokenFinalityDepthV1      = 6

	minerTokenBlockPendingV1   = "PENDING"
	minerTokenBlockCommittedV1 = "COMMITTED"
)

var (
	ErrMinerTokenCanonicalChainRequiredV1 = errors.New(
		"token/miner: canonical blockchain is required",
	)
	ErrMinerTokenBlockIdentityV1 = errors.New(
		"token/miner: invalid block identity",
	)
	ErrMinerTokenFinalizedHeightConflictV1 = errors.New(
		"token/miner: finalized height conflicts with canonical block hash",
	)
	ErrMinerTokenPersistenceV1 = errors.New(
		"token/miner: atomic registry persistence failed",
	)
)

type minerTokenBlockRecordV1 struct {
	Version   string `json:"version"`
	Height    int    `json:"height"`
	BlockHash string `json:"block_hash"`
	Miner     string `json:"miner"`
	Reward    uint64 `json:"reward"`
	Status    string `json:"status"`
}

func minerTokenBlockKeyV1(height int) string {
	return strconv.Itoa(height)
}

func (b *MinerTokenBridge) observeMinerTokenRecordV1(
	outcome string,
	record minerTokenBlockRecordV1,
	confirmationDepth int,
	operationErr error,
) {
	if b == nil {
		return
	}

	observeMinerTokenRewardV1(
		outcome,
		b.cfg.Symbol,
		record.Height,
		record.BlockHash,
		record.Miner,
		record.Reward,
		confirmationDepth,
		operationErr,
	)
}

func (b *MinerTokenBridge) processMinerTokenBlockV1(
	blk *blockchain.Block,
	status miner.MiningStatus,
) error {
	if b == nil || b.chain == nil {
		return ErrMinerTokenCanonicalChainRequiredV1
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if blk == nil ||
		blk.Index <= 0 ||
		len(blk.Hash) != sha256.Size ||
		!validTokenAddress(blk.Miner) {
		err := ErrMinerTokenBlockIdentityV1
		b.observeMinerTokenRecordV1(
			"REJECTED",
			minerTokenBlockRecordV1{},
			0,
			err,
		)
		return err
	}

	record := minerTokenBlockRecordV1{
		Version:   MinerTokenBridgeStateVersionV1,
		Height:    blk.Index,
		BlockHash: hex.EncodeToString(blk.Hash),
		Miner:     blk.Miner,
		Reward:    b.cfg.RewardPerBlock,
		Status:    minerTokenBlockPendingV1,
	}

	if status.BlockHeight != blk.Index ||
		!bytes.Equal(status.BlockHash, blk.Hash) {
		err := ErrMinerTokenBlockIdentityV1
		b.observeMinerTokenRecordV1(
			"REJECTED",
			record,
			0,
			err,
		)
		return err
	}

	canonical := b.chain.BlockByIndexSnapshot(blk.Index)
	if canonical == nil ||
		!bytes.Equal(canonical.Hash, blk.Hash) ||
		canonical.Miner != blk.Miner {
		err := ErrMinerTokenBlockIdentityV1
		b.observeMinerTokenRecordV1(
			"REORG_REJECTED",
			record,
			0,
			err,
		)
		return err
	}

	outcome, err := b.reg.recordPendingMinerTokenBlockV1(
		b.cfg.SaveDir,
		b.cfg.Symbol,
		record,
	)
	if err != nil {
		rejectedOutcome := "REJECTED"
		if errors.Is(err, ErrMinerTokenPersistenceV1) {
			rejectedOutcome = "PERSISTENCE_REJECTED"
		}
		b.observeMinerTokenRecordV1(
			rejectedOutcome,
			record,
			b.chain.BestHeightSnapshot()-record.Height,
			err,
		)
		return err
	}
	b.observeMinerTokenRecordV1(
		outcome,
		record,
		b.chain.BestHeightSnapshot()-record.Height,
		nil,
	)

	return b.reconcilePendingMinerTokenBlocksV1()
}

func (b *MinerTokenBridge) reconcilePendingMinerTokenBlocksV1() error {
	if b == nil || b.chain == nil {
		return ErrMinerTokenCanonicalChainRequiredV1
	}

	tipHeight := b.chain.BestHeightSnapshot()
	records := sortedMinerTokenBlockRecordsV1(
		b.tok.minerTokenBlocksSnapshotV1(),
	)

	for _, record := range records {
		if record.Status != minerTokenBlockPendingV1 {
			continue
		}

		canonical := b.chain.BlockByIndexSnapshot(record.Height)
		if canonical == nil ||
			hex.EncodeToString(canonical.Hash) != record.BlockHash ||
			canonical.Miner != record.Miner {
			if err := b.reg.dropPendingMinerTokenBlockV1(
				b.cfg.SaveDir,
				b.cfg.Symbol,
				record,
			); err != nil {
				b.observeMinerTokenRecordV1(
					"PERSISTENCE_REJECTED",
					record,
					tipHeight-record.Height,
					err,
				)
				return err
			}
			b.observeMinerTokenRecordV1(
				"REORG_REJECTED",
				record,
				tipHeight-record.Height,
				nil,
			)
			continue
		}

		if tipHeight-record.Height < MinerTokenFinalityDepthV1 {
			continue
		}

		committed, err := b.reg.finalizeMinerTokenBlockV1(
			b.cfg.SaveDir,
			b.cfg.Symbol,
			b.cfg.Owner,
			record,
		)
		if err != nil {
			outcome := "REJECTED"
			if errors.Is(err, ErrMinerTokenPersistenceV1) {
				outcome = "PERSISTENCE_REJECTED"
			}
			b.observeMinerTokenRecordV1(
				outcome,
				record,
				tipHeight-record.Height,
				err,
			)
			return err
		}
		if committed {
			b.observeMinerTokenRecordV1(
				"FINALIZED",
				record,
				tipHeight-record.Height,
				nil,
			)
		}
	}

	return nil
}

func (t *Token) minerTokenBlocksSnapshotV1() map[string]minerTokenBlockRecordV1 {
	if t == nil {
		return make(map[string]minerTokenBlockRecordV1)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	return cloneMinerTokenBlockRecordsV1(
		t.minerTokenBlocks,
	)
}

func sortedMinerTokenBlockRecordsV1(
	records map[string]minerTokenBlockRecordV1,
) []minerTokenBlockRecordV1 {
	ordered := make(
		[]minerTokenBlockRecordV1,
		0,
		len(records),
	)

	for _, record := range records {
		ordered = append(ordered, record)
	}

	sort.Slice(
		ordered,
		func(left int, right int) bool {
			return ordered[left].Height < ordered[right].Height
		},
	)

	return ordered
}

type minerTokenMutationV1 func(*Token) (bool, error)

func (r *Registry) persistMinerTokenMutationV1(
	directory string,
	symbol string,
	mutate minerTokenMutationV1,
) error {
	if r == nil {
		return errors.New("token/miner: nil registry")
	}
	if mutate == nil {
		return errors.New("token/miner: nil state mutation")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.ensure()

	tok, exists := r.Tokens[symbol]
	if !exists || tok == nil {
		return fmt.Errorf(
			"token/miner: token %q is not registered",
			symbol,
		)
	}

	tok.mu.Lock()
	defer tok.mu.Unlock()

	before := tokenPersistenceSnapshotLockedV1(tok)

	changed, err := mutate(tok)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}

	data, err := marshalRegistryWithLockedTokenV1(
		r,
		tok,
	)
	if err != nil {
		restoreTokenPersistenceLockedV1(tok, before)
		return fmt.Errorf(
			"%w: encode registry: %v",
			ErrMinerTokenPersistenceV1,
			err,
		)
	}

	if err := writeMinerTokenRegistryV1(
		directory,
		data,
	); err != nil {
		restoreTokenPersistenceLockedV1(tok, before)
		return fmt.Errorf(
			"%w: %v",
			ErrMinerTokenPersistenceV1,
			err,
		)
	}

	return nil
}

func marshalRegistryWithLockedTokenV1(
	r *Registry,
	lockedToken *Token,
) ([]byte, error) {
	tokens := make(
		map[string]json.RawMessage,
		len(r.Tokens),
	)

	for symbol, tok := range r.Tokens {
		if tok == nil {
			return nil, fmt.Errorf(
				"token registry: nil token for %q",
				symbol,
			)
		}

		var (
			encoded []byte
			err     error
		)

		if tok == lockedToken {
			encoded, err = json.Marshal(
				tokenPersistenceSnapshotLockedV1(tok),
			)
		} else {
			encoded, err = json.Marshal(tok)
		}
		if err != nil {
			return nil, err
		}

		tokens[symbol] = json.RawMessage(encoded)
	}

	return json.MarshalIndent(
		struct {
			Tokens map[string]json.RawMessage `json:"tokens"`
		}{
			Tokens: tokens,
		},
		"",
		"  ",
	)
}

func writeMinerTokenRegistryV1(
	directory string,
	data []byte,
) error {
	path := registryFile
	if directory != "" {
		if err := os.MkdirAll(
			directory,
			0o755,
		); err != nil {
			return err
		}

		path = filepath.Join(
			directory,
			registryFile,
		)
	}

	temporaryPath := path + ".miner-token.tmp"
	if err := os.WriteFile(
		temporaryPath,
		data,
		0o644,
	); err != nil {
		return err
	}

	if err := os.Rename(
		temporaryPath,
		path,
	); err != nil {
		_ = os.Remove(temporaryPath)
		return err
	}

	return nil
}

func (r *Registry) recordPendingMinerTokenBlockV1(
	directory string,
	symbol string,
	record minerTokenBlockRecordV1,
) (string, error) {
	outcome := "PENDING_RECORDED"
	key := minerTokenBlockKeyV1(record.Height)

	err := r.persistMinerTokenMutationV1(
		directory,
		symbol,
		func(tok *Token) (bool, error) {
			if tok.minerTokenBlocks == nil {
				tok.minerTokenBlocks =
					make(map[string]minerTokenBlockRecordV1)
			}

			existing, exists := tok.minerTokenBlocks[key]
			if exists {
				if existing.BlockHash == record.BlockHash {
					if existing.Status == minerTokenBlockCommittedV1 {
						outcome = "DUPLICATE_COMMITTED"
					} else {
						outcome = "DUPLICATE_PENDING"
					}
					return false, nil
				}

				if existing.Status == minerTokenBlockCommittedV1 {
					return false,
						ErrMinerTokenFinalizedHeightConflictV1
				}

				outcome = "PENDING_REORG_REPLACED"
			}

			record.Version = MinerTokenBridgeStateVersionV1
			record.Status = minerTokenBlockPendingV1
			tok.minerTokenBlocks[key] = record
			return true, nil
		},
	)

	return outcome, err
}

func (r *Registry) dropPendingMinerTokenBlockV1(
	directory string,
	symbol string,
	record minerTokenBlockRecordV1,
) error {
	key := minerTokenBlockKeyV1(record.Height)

	return r.persistMinerTokenMutationV1(
		directory,
		symbol,
		func(tok *Token) (bool, error) {
			existing, exists := tok.minerTokenBlocks[key]
			if !exists ||
				existing.Status != minerTokenBlockPendingV1 ||
				existing.BlockHash != record.BlockHash {
				return false, nil
			}

			delete(tok.minerTokenBlocks, key)
			return true, nil
		},
	)
}

func (r *Registry) finalizeMinerTokenBlockV1(
	directory string,
	symbol string,
	owner string,
	record minerTokenBlockRecordV1,
) (bool, error) {
	committed := false
	key := minerTokenBlockKeyV1(record.Height)

	err := r.persistMinerTokenMutationV1(
		directory,
		symbol,
		func(tok *Token) (bool, error) {
			existing, exists := tok.minerTokenBlocks[key]
			if !exists {
				return false, ErrMinerTokenBlockIdentityV1
			}
			if existing.BlockHash != record.BlockHash {
				return false,
					ErrMinerTokenFinalizedHeightConflictV1
			}
			if existing.Status == minerTokenBlockCommittedV1 {
				return false, nil
			}
			if existing.Status != minerTokenBlockPendingV1 {
				return false, ErrMinerTokenBlockIdentityV1
			}

			if tok.owner == "" {
				return false, ErrTokenOwnerNotConfigured
			}
			if tok.owner != owner {
				return false, errors.New("not token owner")
			}
			if err := validateMintCommon(
				record.Miner,
				record.Reward,
				tok.totalSupply,
				tok.maxSupply,
			); err != nil {
				return false, err
			}

			newBalance, err := checkedAddUint64(
				tok.balances[record.Miner],
				record.Reward,
			)
			if err != nil {
				return false, ErrTokenBalanceOverflow
			}

			newSupply, err := checkedAddUint64(
				tok.totalSupply,
				record.Reward,
			)
			if err != nil {
				return false, err
			}

			tok.balances[record.Miner] = newBalance
			tok.totalSupply = newSupply
			existing.Status = minerTokenBlockCommittedV1
			tok.minerTokenBlocks[key] = existing
			committed = true
			return true, nil
		},
	)

	return committed, err
}
