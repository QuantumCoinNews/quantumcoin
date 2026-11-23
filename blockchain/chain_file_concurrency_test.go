package blockchain

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"quantumcoin/config"
	"quantumcoin/wallet"
)

func TestChainPersistenceLocksEncodingAndKeepsPreviousFileOnFailure(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "chain.dat")
	if err := os.WriteFile(filename, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	entered, release, second := make(chan struct{}), make(chan struct{}), make(chan struct{})
	results := make(chan error, 2)
	go func() {
		results <- persistChainFileV1(filename, func(w io.Writer) error {
			if _, err := io.WriteString(w, "first-"); err != nil {
				return err
			}
			close(entered)
			<-release
			_, err := io.WriteString(w, "complete")
			return err
		})
	}()
	<-entered
	go func() {
		results <- persistChainFileV1(filepath.Join(filepath.Dir(filename), ".", filepath.Base(filename)), func(w io.Writer) error {
			close(second)
			_, err := io.WriteString(w, "second-complete")
			return err
		})
	}()
	select {
	case <-second:
		t.Error("second encoder ran while first writer owned the destination")
	case <-time.After(30 * time.Millisecond):
	}
	data, err := readChainFileV1(filename)
	if err != nil || string(data) != "original" {
		t.Errorf("partial temporary data became visible: %q %v", data, err)
	}
	close(release)
	for i := 0; i < 2; i++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	data, err = readChainFileV1(filename)
	if err != nil || string(data) != "second-complete" {
		t.Fatalf("unordered install: %q %v", data, err)
	}
	err = persistChainFileV1(filename, func(w io.Writer) error { _, _ = io.WriteString(w, "broken"); return io.ErrUnexpectedEOF })
	if err == nil {
		t.Fatal("encoder failure ignored")
	}
	data, err = readChainFileV1(filename)
	if err != nil || string(data) != "second-complete" {
		t.Fatalf("failed write replaced existing file: %q %v", data, err)
	}
	leftovers, err := filepath.Glob(filepath.Join(filepath.Dir(filename), ".qc-chain-commit-*"))
	if err != nil || len(leftovers) != 0 {
		t.Fatalf("temporary files leaked: %v %v", leftovers, err)
	}
}

func TestChainFileOSLockChildV1(t *testing.T) {
	filename := os.Getenv("QC_TEST_CHAIN_LOCK_CHILD")
	if filename == "" {
		return
	}
	fmt.Fprintln(os.Stdout, "ready")
	if err := persistChainFileV1(filename, func(w io.Writer) error { _, err := io.WriteString(w, "child-complete"); return err }); err != nil {
		t.Fatal(err)
	}
}

func TestChainPersistenceSerializesAnotherProcess(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "process.dat")
	absolute, key, err := canonicalChainFilenameV1(filename)
	if err != nil {
		t.Fatal(err)
	}
	unlock, err := lockChainFileOSV1(absolute, key)
	if err != nil {
		t.Fatal(err)
	}
	locked := true
	defer func() {
		if locked {
			unlock()
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestChainFileOSLockChildV1$")
	command.Env = append(os.Environ(), "QC_TEST_CHAIN_LOCK_CHILD="+filename)
	pipe, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	ready := make(chan string, 1)
	go func() { line, _ := bufio.NewReader(pipe).ReadString('\n'); ready <- line }()
	select {
	case line := <-ready:
		if line != "ready\n" {
			t.Fatalf("child did not reach persistence: %q %s", line, stderr.String())
		}
	case <-ctx.Done():
		t.Fatal("child startup timed out")
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	select {
	case err := <-done:
		t.Fatalf("child bypassed OS writer lock: %v %s", err, stderr.String())
	case <-time.After(75 * time.Millisecond):
	}
	unlock()
	locked = false
	if err := <-done; err != nil {
		t.Fatalf("child save failed: %v %s", err, stderr.String())
	}
	data, err := readChainFileV1(filename)
	if err != nil || string(data) != "child-complete" {
		t.Fatalf("child file invalid: %q %v", data, err)
	}
}

func TestConcurrentSavesMiningAndTrustedReadersProduceValidFiles(t *testing.T) {
	t.Setenv("QC_DIFFICULTY_BITS", "8")
	chain := NewBlockchain(config.Default().InitialReward, config.Default().TotalSupply)
	filename := filepath.Join(t.TempDir(), "chain.dat")
	t.Setenv("QC_NODE_DIR", t.TempDir())
	if err := chain.SaveToFile(filename); err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	failures := make(chan error, 100)
	for i := 0; i < 4; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 4; j++ {
				if err := chain.SaveToFile(filename); err != nil {
					failures <- err
				}
			}
		}()
	}
	for i := 0; i < 3; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 8; j++ {
				if _, err := LoadBlockchainFromFileWithLocalContextV1(filename, chain); err != nil {
					failures <- err
				}
			}
		}()
	}
	workers.Add(1)
	go func() {
		defer workers.Done()
		bits, err := chain.ExpectedNextDifficultyV1()
		if err == nil {
			_, err = chain.MineBlock(wallet.NewWallet().GetAddress(), bits)
		}
		if err != nil {
			failures <- err
			return
		}
		if err := chain.SaveToFile(filename); err != nil {
			failures <- err
		}
	}()
	workers.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	loaded, err := LoadBlockchainFromFileWithLocalContextV1(filename, chain)
	if err != nil || loaded.BestHeightSnapshot() != 1 {
		t.Fatalf("new mined state lost: %v", err)
	}
}

func TestNFTCommitAndNormalSavesShareOnePersistenceBoundary(t *testing.T) {
	f := nftDurableMiningFixtureV1(t)
	var workers sync.WaitGroup
	failures := make(chan error, 20)
	for i := 0; i < 3; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 3; j++ {
				if err := f.chain.SaveToFile(f.filename); err != nil {
					failures <- err
				}
			}
		}()
	}
	workers.Add(1)
	go func() {
		defer workers.Done()
		_, err := f.chain.MineBlockWithNFTAwardAndPersistSynchronizedV1(wallet.NewWallet().GetAddress(), f.record, f.proof, f.filename)
		if err != nil {
			failures <- err
		}
	}()
	workers.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	loaded, err := LoadBlockchainFromFileWithLocalContextV1(f.filename, f.chain)
	if err != nil {
		t.Fatal(err)
	}
	ownership, err := loaded.NFTOwnershipSnapshotSynchronizedV1()
	if err != nil || len(ownership.NFTs) != 1 {
		t.Fatalf("normal save overwrote NFT commit: %+v %v", ownership, err)
	}
}
