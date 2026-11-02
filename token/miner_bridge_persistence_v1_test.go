package token

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func persistentMinerBridgeTestRecord(
	height int,
) minerTokenBlockRecordV1 {
	hash := sha256.Sum256([]byte("persistent-record"))

	return minerTokenBlockRecordV1{
		Version:   MinerTokenBridgeStateVersionV1,
		Height:    height,
		BlockHash: hex.EncodeToString(hash[:]),
		Miner:     canonicalAddressFixture("miner"),
		Reward:    25,
		Status:    minerTokenBlockPendingV1,
	}
}

func TestMinerTokenBridgeV1CommittedStateSurvivesRestart(
	t *testing.T,
) {
	directory := t.TempDir()
	chain := minerBridgeTestChain(7, "miner", "canonical")
	bridge := minerBridgeTestBridge(t, chain, directory)
	block := chain.GetBlockByIndex(1)
	status := minerBridgeTestStatus(block)

	if err := bridge.processMinerTokenBlockV1(
		block,
		status,
	); err != nil {
		t.Fatalf("initial finalized reward: %v", err)
	}
	if got := bridge.tok.TotalSupply(); got != 25 {
		t.Fatalf("initial supply = %d", got)
	}

	restarted := minerBridgeTestBridge(t, chain, directory)
	if err := restarted.processMinerTokenBlockV1(
		block,
		status,
	); err != nil {
		t.Fatalf("restart duplicate callback: %v", err)
	}
	if got := restarted.tok.TotalSupply(); got != 25 {
		t.Fatalf("restart supply = %d, want 25", got)
	}
}

func TestMinerTokenBridgeV1PendingStateSurvivesRestart(
	t *testing.T,
) {
	directory := t.TempDir()
	chain := minerBridgeTestChain(1, "miner", "canonical")
	bridge := minerBridgeTestBridge(t, chain, directory)
	block := chain.GetBlockByIndex(1)

	if err := bridge.processMinerTokenBlockV1(
		block,
		minerBridgeTestStatus(block),
	); err != nil {
		t.Fatalf("record pending reward: %v", err)
	}

	restarted := minerBridgeTestBridge(t, chain, directory)
	records := restarted.tok.minerTokenBlocksSnapshotV1()
	record, exists := records[minerTokenBlockKeyV1(1)]
	if !exists || record.Status != minerTokenBlockPendingV1 {
		t.Fatalf("pending record after restart = %#v", record)
	}

	appendCanonicalMinerBridgeTestBlock(
		chain,
		7,
		"miner",
		"canonical",
	)
	head := chain.GetBlockByIndex(7)
	if err := restarted.processMinerTokenBlockV1(
		head,
		minerBridgeTestStatus(head),
	); err != nil {
		t.Fatalf("finalize restarted pending reward: %v", err)
	}
	if got := restarted.tok.TotalSupply(); got != 25 {
		t.Fatalf("restarted pending supply = %d, want 25", got)
	}
}

func TestMinerTokenBridgeV1SaveFailureRollsBackPendingState(
	t *testing.T,
) {
	baseDirectory := t.TempDir()
	chain := minerBridgeTestChain(1, "miner", "canonical")
	bridge := minerBridgeTestBridge(t, chain, baseDirectory)
	block := chain.GetBlockByIndex(1)

	blockedPath := filepath.Join(baseDirectory, "not-a-directory")
	if err := os.WriteFile(
		blockedPath,
		[]byte("blocked"),
		0o600,
	); err != nil {
		t.Fatalf("create blocked path: %v", err)
	}
	bridge.cfg.SaveDir = filepath.Join(blockedPath, "child")

	err := bridge.processMinerTokenBlockV1(
		block,
		minerBridgeTestStatus(block),
	)
	if !errors.Is(err, ErrMinerTokenPersistenceV1) {
		t.Fatalf("pending persistence error = %v", err)
	}
	if got := bridge.tok.TotalSupply(); got != 0 {
		t.Fatalf("pending rollback supply = %d", got)
	}
	if got := len(bridge.tok.minerTokenBlocksSnapshotV1()); got != 0 {
		t.Fatalf("pending rollback record count = %d", got)
	}
}

func TestMinerTokenBridgeV1SaveFailureRollsBackFinalization(
	t *testing.T,
) {
	baseDirectory := t.TempDir()
	chain := minerBridgeTestChain(1, "miner", "canonical")
	bridge := minerBridgeTestBridge(t, chain, baseDirectory)
	block := chain.GetBlockByIndex(1)

	if err := bridge.processMinerTokenBlockV1(
		block,
		minerBridgeTestStatus(block),
	); err != nil {
		t.Fatalf("record pending reward: %v", err)
	}

	appendCanonicalMinerBridgeTestBlock(
		chain,
		7,
		"miner",
		"canonical",
	)

	blockedPath := filepath.Join(baseDirectory, "not-a-directory")
	if err := os.WriteFile(
		blockedPath,
		[]byte("blocked"),
		0o600,
	); err != nil {
		t.Fatalf("create blocked path: %v", err)
	}
	bridge.cfg.SaveDir = filepath.Join(blockedPath, "child")

	err := bridge.reconcilePendingMinerTokenBlocksV1()
	if !errors.Is(err, ErrMinerTokenPersistenceV1) {
		t.Fatalf("finalization persistence error = %v", err)
	}
	if got := bridge.tok.TotalSupply(); got != 0 {
		t.Fatalf("finalization rollback supply = %d", got)
	}
	if got := bridge.tok.BalanceOf(canonicalAddressFixture("miner")); got != 0 {
		t.Fatalf("finalization rollback balance = %d", got)
	}
	records := bridge.tok.minerTokenBlocksSnapshotV1()
	record, exists := records[minerTokenBlockKeyV1(1)]
	if !exists || record.Status != minerTokenBlockPendingV1 {
		t.Fatalf("finalization rollback record = %#v", record)
	}
}

func TestMinerTokenBridgeV1RegistryRejectsInvalidPersistentState(
	t *testing.T,
) {
	testCases := []struct {
		name   string
		key    string
		record minerTokenBlockRecordV1
	}{
		{
			name: "invalid version",
			key:  "1",
			record: func() minerTokenBlockRecordV1 {
				record := persistentMinerBridgeTestRecord(1)
				record.Version = "UNKNOWN"
				return record
			}(),
		},
		{
			name: "invalid height key",
			key:  "2",
			record: func() minerTokenBlockRecordV1 {
				return persistentMinerBridgeTestRecord(1)
			}(),
		},
		{
			name: "invalid hash",
			key:  "1",
			record: func() minerTokenBlockRecordV1 {
				record := persistentMinerBridgeTestRecord(1)
				record.BlockHash = "bad"
				return record
			}(),
		},
		{
			name: "duplicate hash",
			key:  "2",
			record: func() minerTokenBlockRecordV1 {
				record := persistentMinerBridgeTestRecord(2)
				first := persistentMinerBridgeTestRecord(1)
				record.BlockHash = first.BlockHash
				return record
			}(),
		},
		{
			name: "invalid miner",
			key:  "1",
			record: func() minerTokenBlockRecordV1 {
				record := persistentMinerBridgeTestRecord(1)
				record.Miner = " "
				return record
			}(),
		},
		{
			name: "zero reward",
			key:  "1",
			record: func() minerTokenBlockRecordV1 {
				record := persistentMinerBridgeTestRecord(1)
				record.Reward = 0
				return record
			}(),
		},
		{
			name: "invalid status",
			key:  "1",
			record: func() minerTokenBlockRecordV1 {
				record := persistentMinerBridgeTestRecord(1)
				record.Status = "UNKNOWN"
				return record
			}(),
		},
	}

	owner := canonicalAddressFixture(
		"owner",
	)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			directory := t.TempDir()
			registry := NewRegistry()
			subject := New(
				"MINE",
				"Mineable Test Token",
				8,
				1_000,
				owner,
			)
			subject.minerTokenBlocks[testCase.key] = testCase.record

			if testCase.name == "duplicate hash" {
				subject.minerTokenBlocks["1"] =
					persistentMinerBridgeTestRecord(1)
			}

			if err := registry.Register(subject); err != nil {
				t.Fatalf("register invalid fixture: %v", err)
			}
			if err := registry.Save(directory); err != nil {
				t.Fatalf("save invalid fixture: %v", err)
			}

			loaded := NewRegistry()
			err := loaded.Load(directory)
			if !errors.Is(err, ErrRegistryLoadIntegrityV1) {
				t.Fatalf("load error = %v, want integrity error", err)
			}
		})
	}
}
