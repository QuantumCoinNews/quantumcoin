package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/wallet"
)

func TestAPIConcurrentReloadAndReadsUseRuntimeSnapshots(t *testing.T) {
	handler := readOnlyMuxFixtureV1(t)
	t.Setenv("QC_NODE_DIR", t.TempDir())
	t.Setenv("QC_DIFFICULTY_BITS", "8")
	initial, _ := nodeRuntime.Snapshot()
	producer := blockchain.DeserializeBlockchain(blockchain.SerializeBlockchain(initial))
	producer.SetCoinbaseMaturity(cfg.CoinbaseMaturity)
	address := wallet.NewWallet().GetAddress()
	filename := cfg.ChainFile
	paths := []string{"/api/telemetry", "/api/health", "/api/blocks", "/api/block?index=0", "/api/security/dashboard", "/api/wallet/balance/" + address}
	var workers sync.WaitGroup
	failures := make(chan error, 100)
	for i := 0; i < 4; i++ {
		workers.Add(1)
		go func(offset int) {
			defer workers.Done()
			for j := 0; j < 18; j++ {
				path := paths[(offset+j)%len(paths)]
				response := readOnlyRequestV1(handler, http.MethodGet, path, actualMuxTestTokenV1)
				// Heavy advisory reads now have explicit rate/concurrency budgets.
				// A bounded rejection is expected under this deliberate burst;
				// every other read and the final validated reload still must pass.
				if path == "/api/security/dashboard" && response.Code == http.StatusTooManyRequests {
					continue
				}
				if response.Code != http.StatusOK {
					failures <- fmt.Errorf("read %s: status=%d body=%s", path, response.Code, response.Body.String())
				}
			}
		}(i)
	}
	workers.Add(1)
	go func() {
		defer workers.Done()
		for i := 0; i < 2; i++ {
			bits, err := producer.ExpectedNextDifficultyV1()
			if err == nil {
				_, err = producer.MineBlock(address, bits)
			}
			if err == nil {
				err = producer.SaveToFile(filename)
			}
			if err != nil {
				failures <- err
				return
			}
		}
	}()
	workers.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	if _, err := nodeRuntime.ReloadChainFromDiskIfChanged(); err != nil {
		t.Fatal(err)
	}
	current, _ := nodeRuntime.Snapshot()
	if current == initial || current.BestHeightSnapshot() != 2 {
		t.Fatalf("observer did not install latest validated chain: height=%d", current.BestHeightSnapshot())
	}
	if err := current.ValidateBlockchainSynchronized(); err != nil {
		t.Fatal(err)
	}
}

func TestAdaptersHaveNoGlobalBlockchainPointerOrDirectChainFileWrites(t *testing.T) {
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		if file.Scope.Lookup("bc") != nil {
			t.Errorf("%s retains a package-global blockchain pointer", path)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if s, ok := call.Fun.(*ast.SelectorExpr); ok && s.Sel.Name == "SaveToFile" {
					t.Errorf("%s bypasses Runtime.SaveChain", path)
				}
			}
			return true
		})
	}
}

func TestCanonicalChainWritersShareOneInstallationBoundary(t *testing.T) {
	for _, entry := range []struct{ path, name string }{
		{"blockchain/blockchain.go", "SaveToFile"},
		{"blockchain/state_mutation.go", "CommitValidatedBlockAndPersist"},
		{"blockchain/nft_durable_mining_v1.go", "persistNFTMiningCandidateLockedV1"},
	} {
		function := runtimePolicyFunctionV1(t, entry.path, entry.name)
		if runtimePolicyFirstCallV1(function, "persistChainFileV1") == token.NoPos {
			t.Errorf("%s bypasses common persistence", entry.name)
		}
		for _, unsafe := range []string{"os.WriteFile", "os.Rename"} {
			if runtimePolicyFirstCallV1(function, unsafe) != token.NoPos {
				t.Errorf("%s uses independent %s", entry.name, unsafe)
			}
		}
	}
}
