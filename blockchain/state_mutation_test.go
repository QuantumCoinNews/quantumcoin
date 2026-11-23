package blockchain

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"quantumcoin/wallet"
)

func TestBlockchainCoinbaseMaturityWriterIsSynchronized(
	t *testing.T,
) {
	chain := newStateSnapshotTestChain()

	const iterations = 1000

	var wait sync.WaitGroup

	wait.Add(2)

	go func() {
		defer wait.Done()

		for index := 0; index < iterations; index++ {
			chain.SetCoinbaseMaturity(
				index % 100,
			)
		}
	}()

	go func() {
		defer wait.Done()

		for index := 0; index < iterations; index++ {
			snapshot := chain.StateSnapshot()

			if snapshot.CoinbaseMaturity < 0 {
				t.Errorf(
					"negative coinbase maturity: %d",
					snapshot.CoinbaseMaturity,
				)
				return
			}
		}
	}()

	wait.Wait()
}

func TestCoreMutationEntrypointsUseCanonicalWriteLock(
	t *testing.T,
) {
	source := readBlockchainSourceForMutationTest(
		t,
		"blockchain.go",
	)

	for _, functionName := range []string{
		"AddBlock",
		"addBlockFromPeerAfterStrictV2",
		"AddTransaction",
		"replaceChainAfterStrictV1",
		"SetCoinbaseMaturity",
	} {
		body := blockchainFunctionBodyForMutationTest(
			t,
			source,
			functionName,
		)

		if !containsAllForMutationTest(
			body,
			"bc.stateMu.Lock()",
			"bc.stateMu.Unlock()",
		) {
			t.Fatalf(
				"%s does not own canonical write lock",
				functionName,
			)
		}
	}
}

// Mining wrappers delegate lock acquisition to context-aware entry points.
// Verify exclusion through actual calls rather than requiring lock text in
// the wrapper: none may publish a candidate while a writer owns the state.
func TestMiningEntryPointsWaitForCanonicalWriteLock(t *testing.T) {
	for _, name := range []string{"memory", "memory-context", "persistent", "persistent-context"} {
		t.Run(name, func(t *testing.T) {
			chain := NewIsolatedLocalBlockchainV1(50, 0)
			address := wallet.NewWallet().GetAddress()
			file := filepath.Join(t.TempDir(), "chain.dat")
			chain.stateMu.Lock()
			locked := true
			defer func() {
				if locked {
					chain.stateMu.Unlock()
				}
			}()
			started, done := make(chan struct{}), make(chan error, 1)
			go func() {
				close(started)
				var err error
				switch name {
				case "memory":
					_, err = chain.MineBlock(address, 1)
				case "memory-context":
					_, err = chain.MineBlockContextV1(context.Background(), address, 1)
				case "persistent":
					_, err = chain.MineBlockAndPersistV1(address, 1, file)
				case "persistent-context":
					_, err = chain.MineBlockAndPersistContextV1(context.Background(), address, 1, file)
				}
				done <- err
			}()
			<-started
			select {
			case err := <-done:
				t.Fatal("mining bypassed the active state writer", err)
			case <-time.After(30 * time.Millisecond):
			}
			chain.stateMu.Unlock()
			locked = false
			select {
			case err := <-done:
				if err != nil || chain.BestHeightSnapshot() != 1 {
					t.Fatal("mining did not commit after writer released state", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("mining did not resume after writer released state")
			}
		})
	}
}

func readBlockchainSourceForMutationTest(
	t *testing.T,
	name string,
) string {
	t.Helper()

	data, err := osReadFileForMutationTest(name)
	if err != nil {
		t.Fatalf(
			"read %s: %v",
			name,
			err,
		)
	}

	return string(data)
}

func blockchainFunctionBodyForMutationTest(
	t *testing.T,
	source string,
	name string,
) string {
	t.Helper()

	startPattern := "func (bc *Blockchain) " + name

	start := stringsIndexForMutationTest(
		source,
		startPattern,
	)

	if start < 0 {
		t.Fatalf(
			"function missing: %s",
			name,
		)
	}

	openRelative := stringsIndexForMutationTest(
		source[start:],
		"{",
	)

	if openRelative < 0 {
		t.Fatalf(
			"function body missing: %s",
			name,
		)
	}

	open := start + openRelative

	depth := 0

	for index := open; index < len(source); index++ {
		switch source[index] {
		case '{':
			depth++
		case '}':
			depth--

			if depth == 0 {
				return source[start : index+1]
			}
		}
	}

	t.Fatalf(
		"unterminated function: %s",
		name,
	)

	return ""
}

func containsAllForMutationTest(
	source string,
	values ...string,
) bool {
	for _, value := range values {
		if stringsIndexForMutationTest(
			source,
			value,
		) < 0 {
			return false
		}
	}

	return true
}
func osReadFileForMutationTest(
	name string,
) ([]byte, error) {
	return os.ReadFile(name)
}

func stringsIndexForMutationTest(
	source string,
	value string,
) int {
	return strings.Index(
		source,
		value,
	)
}
