package node

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/wallet"
)

func TestLegacyMiningCancellationIsProcessBounded(t *testing.T) {
	if mode := os.Getenv("QC_TEST_LEGACY_CANCEL_CHILD_V1"); mode != "" {
		chain := blockchain.NewBlockchain(50, 0)
		if mode == "isolated-local" {
			chain = blockchain.NewIsolatedLocalBlockchainV1(50, 0)
		}
		file := filepath.Join(os.Getenv("QC_TEST_LEGACY_CANCEL_DIR"), "cancel-chain.dat")
		if err := chain.SaveToFile(file); err != nil {
			t.Fatal(err)
		}
		before, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		r := NewRuntime(chain, &config.Config{ChainFile: file})
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
		defer cancel()
		block, err := r.MineOnceContextV1(ctx, wallet.NewWallet().GetAddress(), 255)
		if block != nil || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("canceled search returned block or wrong error: %v", err)
		}
		after, err := os.ReadFile(file)
		if err != nil || !bytes.Equal(before, after) || chain.BestHeightSnapshot() != 0 {
			t.Fatal("canceled search changed persistent or live chain")
		}
		fmt.Println("LEGACY_CANCEL_OK")
		return
	}
	for _, mode := range []string{"legacy", "isolated-local"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestLegacyMiningCancellationIsProcessBounded$")
			command.Env = append(os.Environ(), "QC_TEST_LEGACY_CANCEL_CHILD_V1="+mode, "QC_TEST_LEGACY_CANCEL_DIR="+t.TempDir())
			output, err := command.CombinedOutput()
			if err != nil || !strings.Contains(string(output), "LEGACY_CANCEL_OK") {
				t.Fatalf("%s cancellation did not finish; child stopped by watchdog: %v", mode, err)
			}
		})
	}
}
