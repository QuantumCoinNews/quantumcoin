package token

import (
	"bytes"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/miner"
)

func TestMinerTokenBridgeErrorPropagationV1CallsOnErrorOnce(
	t *testing.T,
) {
	chain := minerBridgeTestChain(1, "miner", "canonical")
	bridge := minerBridgeTestBridge(t, chain, t.TempDir())
	block := chain.GetBlockByIndex(1)
	status := minerBridgeTestStatus(block)
	status.BlockHeight++

	errorCount := 0
	onBlockCount := 0
	var callbackErr error

	opts := bridge.Options(miner.Options{
		OnError: func(err error) {
			errorCount++
			callbackErr = err
		},
		OnBlock: func(
			_ *blockchain.Block,
			_ miner.MiningStatus,
		) {
			onBlockCount++
		},
	})

	opts.OnBlock(block, status)

	if errorCount != 1 {
		t.Fatalf("error callback count = %d, want 1", errorCount)
	}
	if !errors.Is(callbackErr, ErrMinerTokenBlockIdentityV1) {
		t.Fatalf("callback error = %v", callbackErr)
	}
	if onBlockCount != 1 {
		t.Fatalf("user OnBlock count = %d, want 1", onBlockCount)
	}
}

func TestMinerTokenBridgeErrorPropagationV1SuccessIsSilent(
	t *testing.T,
) {
	chain := minerBridgeTestChain(1, "miner", "canonical")
	bridge := minerBridgeTestBridge(t, chain, t.TempDir())
	block := chain.GetBlockByIndex(1)

	errorCount := 0
	onBlockCount := 0

	opts := bridge.Options(miner.Options{
		OnError: func(error) {
			errorCount++
		},
		OnBlock: func(
			_ *blockchain.Block,
			_ miner.MiningStatus,
		) {
			onBlockCount++
		},
	})

	opts.OnBlock(block, minerBridgeTestStatus(block))

	if errorCount != 0 {
		t.Fatalf("success error count = %d", errorCount)
	}
	if onBlockCount != 1 {
		t.Fatalf("success OnBlock count = %d", onBlockCount)
	}
}

func TestMinerTokenBridgeErrorPropagationV1PersistenceRollback(
	t *testing.T,
) {
	baseDirectory := t.TempDir()
	chain := minerBridgeTestChain(1, "miner", "canonical")
	bridge := minerBridgeTestBridge(t, chain, baseDirectory)
	block := chain.GetBlockByIndex(1)

	blockedPath := filepath.Join(
		baseDirectory,
		"not-a-directory",
	)
	if err := os.WriteFile(
		blockedPath,
		[]byte("blocked"),
		0o600,
	); err != nil {
		t.Fatalf("create blocked path: %v", err)
	}

	bridge.cfg.SaveDir = filepath.Join(
		blockedPath,
		"child",
	)

	errorCount := 0
	var callbackErr error

	opts := bridge.Options(miner.Options{
		OnError: func(err error) {
			errorCount++
			callbackErr = err
		},
	})

	opts.OnBlock(block, minerBridgeTestStatus(block))

	if errorCount != 1 {
		t.Fatalf("persistence error count = %d, want 1", errorCount)
	}
	if !errors.Is(callbackErr, ErrMinerTokenPersistenceV1) {
		t.Fatalf("persistence callback error = %v", callbackErr)
	}
	if got := bridge.tok.TotalSupply(); got != 0 {
		t.Fatalf("rollback total supply = %d", got)
	}
	if got := len(
		bridge.tok.minerTokenBlocksSnapshotV1(),
	); got != 0 {
		t.Fatalf("rollback record count = %d", got)
	}
}

func TestMinerTokenBridgeErrorPropagationV1SharesMinerHandler(
	t *testing.T,
) {
	chain := minerBridgeTestChain(1, "miner", "canonical")
	bridge := minerBridgeTestBridge(t, chain, t.TempDir())
	sentinel := errors.New("miner failure")

	var received error

	opts := bridge.Options(miner.Options{
		OnError: func(err error) {
			received = err
		},
	})

	if opts.OnError == nil {
		t.Fatal("miner OnError handler was not preserved")
	}

	opts.OnError(sentinel)

	if !errors.Is(received, sentinel) {
		t.Fatalf("miner callback error = %v", received)
	}
}

func TestMinerTokenBridgeErrorPropagationV1FallbackLog(
	t *testing.T,
) {
	chain := minerBridgeTestChain(1, "miner", "canonical")
	bridge := minerBridgeTestBridge(t, chain, t.TempDir())
	block := chain.GetBlockByIndex(1)
	status := minerBridgeTestStatus(block)
	status.BlockHeight++

	previousWriter := log.Writer()
	previousFlags := log.Flags()
	previousPrefix := log.Prefix()

	var output bytes.Buffer

	log.SetOutput(&output)
	log.SetFlags(0)
	log.SetPrefix("")

	t.Cleanup(func() {
		log.SetOutput(previousWriter)
		log.SetFlags(previousFlags)
		log.SetPrefix(previousPrefix)
	})

	bridge.Options().OnBlock(block, status)

	if !strings.Contains(
		output.String(),
		"token/miner: bridge callback failed",
	) {
		t.Fatalf("fallback log = %q", output.String())
	}
}
