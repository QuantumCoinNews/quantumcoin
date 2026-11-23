package token

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/miner"
	"quantumcoin/security"
)

func bridgeLedgerBytes(t *testing.T, bridge *MinerTokenBridge) ([]byte, []byte) {
	t.Helper()
	memory, err := json.Marshal(bridge.reg)
	if err != nil {
		t.Fatal(err)
	}
	disk, err := os.ReadFile(filepath.Join(bridge.cfg.SaveDir, registryFile))
	if err != nil {
		t.Fatal(err)
	}
	return memory, disk
}

func TestMinerBridgeActualWriteFailureRollsBackAndRetriesOnce(t *testing.T) {
	for _, stage := range []string{"pending", "finalization"} {
		t.Run(stage, func(t *testing.T) {
			dir := t.TempDir()
			chain := minerBridgeTestChain(1, "miner", "canonical")
			bridge := minerBridgeTestBridge(t, chain, dir)
			block := chain.GetBlockByIndex(1)
			if stage == "finalization" {
				if err := bridge.processMinerTokenBlockV1(block, minerBridgeTestStatus(block)); err != nil {
					t.Fatal(err)
				}
				appendCanonicalMinerBridgeTestBlock(chain, 7, "miner", "canonical")
			}
			beforeMemory, beforeDisk := bridgeLedgerBytes(t, bridge)
			// A directory at the staging-file path fails os.WriteFile after
			// lock acquisition, disk verification and the in-memory mutation.
			obstacle := filepath.Join(dir, registryFile+".miner-token.tmp")
			if err := os.Mkdir(obstacle, 0700); err != nil {
				t.Fatal(err)
			}
			if stage == "finalization" {
				record := bridge.tok.minerTokenBlocksSnapshotV1()["1"]
				committed, err := bridge.reg.finalizeMinerTokenBlockV1(dir, bridge.cfg.Symbol, bridge.cfg.Owner, record)
				if committed || !errors.Is(err, ErrMinerTokenPersistenceV1) {
					t.Fatalf("failed commit reported committed=%v, err=%v", committed, err)
				}
			}
			errorsSeen, blocksSeen := 0, 0
			var callbackErr error
			opts := bridge.Options(miner.Options{
				OnError: func(err error) { errorsSeen++; callbackErr = err },
				OnBlock: func(*blockchain.Block, miner.MiningStatus) { blocksSeen++ },
			})
			opts.OnBlock(block, minerBridgeTestStatus(block))
			if errorsSeen != 1 || blocksSeen != 1 || !errors.Is(callbackErr, ErrMinerTokenPersistenceV1) {
				t.Fatalf("error=%v counts error/block=%d/%d", callbackErr, errorsSeen, blocksSeen)
			}
			var pathErr *os.PathError
			if !errors.As(callbackErr, &pathErr) || pathErr.Path != obstacle {
				t.Fatalf("original OS write error was lost: %v", callbackErr)
			}
			afterMemory, afterDisk := bridgeLedgerBytes(t, bridge)
			if !bytes.Equal(beforeMemory, afterMemory) || !bytes.Equal(beforeDisk, afterDisk) {
				t.Fatal("failed write changed memory or durable ledger")
			}
			for _, event := range security.ReadEvents(dir, 100).Events {
				if event.Type == TokenMinerRewardFinalizedEventV1 {
					t.Fatal("failed write announced a finalized token reward")
				}
			}
			if chain.BlockByIndexSnapshot(1).Miner != block.Miner {
				t.Fatal("secondary token failure changed canonical QC block")
			}
			// Remove only this test's empty obstacle, then retry the same job.
			if err := os.Remove(obstacle); err != nil {
				t.Fatal(err)
			}
			opts.OnBlock(block, minerBridgeTestStatus(block))
			if stage == "pending" {
				appendCanonicalMinerBridgeTestBlock(chain, 7, "miner", "canonical")
				opts.OnBlock(block, minerBridgeTestStatus(block))
			}
			opts.OnBlock(block, minerBridgeTestStatus(block))
			if errorsSeen != 1 || bridge.tok.TotalSupply() != 25 || bridge.tok.BalanceOf(block.Miner) != 25 {
				t.Fatal("retry produced another error or duplicate reward")
			}
			restarted := minerBridgeTestBridge(t, chain, dir)
			if err := restarted.processMinerTokenBlockV1(block, minerBridgeTestStatus(block)); err != nil {
				t.Fatal(err)
			}
			if restarted.tok.TotalSupply() != 25 {
				t.Fatal("restart after retry minted again")
			}
		})
	}
}

func TestMinerBridgeMutationFailureRestoresPartialChanges(t *testing.T) {
	for _, result := range []string{"error", "unchanged"} {
		t.Run(result, func(t *testing.T) {
			bridge := minerBridgeTestBridge(t, minerBridgeTestChain(1, "miner", "canonical"), t.TempDir())
			beforeMemory, beforeDisk := bridgeLedgerBytes(t, bridge)
			sentinel := errors.New("operation failed after partial mutation")
			err := bridge.reg.persistMinerTokenMutationV1(bridge.cfg.SaveDir, bridge.cfg.Symbol, func(tok *Token) (bool, error) {
				tok.totalSupply = 7
				tok.balances[bridge.cfg.Owner] = 7
				tok.authorizationNonces[bridge.cfg.Owner] = 4
				if result == "error" {
					return true, sentinel
				}
				return false, nil
			})
			if result == "error" && !errors.Is(err, sentinel) {
				t.Fatalf("operation error lost: %v", err)
			}
			if result == "unchanged" && err != nil {
				t.Fatal(err)
			}
			afterMemory, afterDisk := bridgeLedgerBytes(t, bridge)
			if !bytes.Equal(beforeMemory, afterMemory) || !bytes.Equal(beforeDisk, afterDisk) {
				t.Fatal("aborted mutation leaked into memory or disk")
			}
		})
	}
}

func TestMinerBridgeMintCapFailureIsReportedWithoutIssuance(t *testing.T) {
	t.Setenv("QC_ENABLE_TOKEN_MINER_BRIDGE", "1")
	dir := t.TempDir()
	t.Setenv("QC_NODE_DIR", dir)
	chain := minerBridgeTestChain(7, "miner", "canonical")
	bridge, err := NewMinerTokenBridge(MinerTokenConfig{Symbol: "CAP", Name: "Capped Token", Owner: canonicalAddressFixture("owner"), MaxSupply: 10, RewardPerBlock: 25, SaveDir: dir}, chain)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	opts := bridge.Options(miner.Options{OnError: func(err error) {
		count++
		if !errors.Is(err, ErrTokenMaxSupplyExceeded) {
			t.Errorf("mint error=%v", err)
		}
	}})
	block := chain.GetBlockByIndex(1)
	for range 2 {
		opts.OnBlock(block, minerBridgeTestStatus(block))
	}
	if count != 2 || bridge.tok.TotalSupply() != 0 || bridge.tok.BalanceOf(block.Miner) != 0 {
		t.Fatal("cap error was hidden or issued tokens")
	}
	if bridge.tok.minerTokenBlocksSnapshotV1()["1"].Status != minerTokenBlockPendingV1 {
		t.Fatal("rejected mint was marked committed")
	}
	for _, event := range security.ReadEvents(dir, 100).Events {
		if event.Type == TokenMinerRewardFinalizedEventV1 {
			t.Fatal("cap rejection announced finalized reward")
		}
	}
}
