package blockchain

import (
	"os"
	"strings"
	"sync"
	"testing"
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
		"MineBlock",
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
