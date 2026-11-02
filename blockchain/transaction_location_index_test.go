package blockchain

import (
	"encoding/hex"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestTransactionLocationSnapshotRebuildsCanonicalIndexes(
	t *testing.T,
) {
	confirmedID := []byte{0x10, 0x01}
	pendingID := []byte{0x20, 0x02}
	bothID := []byte{0x30, 0x03}

	chain := &Blockchain{
		Blocks: []*Block{
			{
				Index: 3,
				Transactions: []*Transaction{
					{ID: confirmedID},
					{ID: bothID},
				},
			},
			{
				Index: 8,
				Transactions: []*Transaction{
					{ID: bothID},
				},
			},
		},
		pendingTxs: []*Transaction{
			{ID: pendingID},
			{ID: bothID},
		},
	}

	confirmed :=
		chain.TransactionLocationSnapshot(
			hex.EncodeToString(confirmedID),
		)

	if !confirmed.InBlock ||
		confirmed.InMempool ||
		confirmed.ConfirmedHeight != 3 {
		t.Fatalf(
			"confirmed=%+v",
			confirmed,
		)
	}

	pending :=
		chain.TransactionLocationSnapshot(
			hex.EncodeToString(pendingID),
		)

	if pending.InBlock ||
		!pending.InMempool ||
		pending.ConfirmedHeight != -1 {
		t.Fatalf(
			"pending=%+v",
			pending,
		)
	}

	both :=
		chain.TransactionLocationSnapshot(
			hex.EncodeToString(bothID),
		)

	if !both.InBlock ||
		!both.InMempool ||
		both.ConfirmedHeight != 3 {
		t.Fatalf(
			"both=%+v",
			both,
		)
	}

	unknown :=
		chain.TransactionLocationSnapshot(
			"ffffffff",
		)

	if unknown.InBlock ||
		unknown.InMempool ||
		unknown.ConfirmedHeight != -1 {
		t.Fatalf(
			"unknown=%+v",
			unknown,
		)
	}
}

func TestTransactionLocationIndexPreservesHistoricalStateOnLazyInit(
	t *testing.T,
) {
	oldID := []byte{0xaa, 0x01}
	newID := []byte{0xbb, 0x02}

	chain := &Blockchain{
		Blocks: []*Block{
			{
				Index: 2,
				Transactions: []*Transaction{
					{ID: oldID},
				},
			},
		},
		pendingTxs: []*Transaction{},
	}

	newBlock := &Block{
		Index: 3,
		Transactions: []*Transaction{
			{ID: newID},
		},
	}

	chain.stateMu.Lock()

	chain.Blocks = append(
		chain.Blocks,
		newBlock,
	)

	chain.indexConfirmedBlockTransactionsLocked(
		newBlock,
	)

	chain.stateMu.Unlock()

	oldLocation :=
		chain.TransactionLocationSnapshot(
			hex.EncodeToString(oldID),
		)

	if !oldLocation.InBlock ||
		oldLocation.ConfirmedHeight != 2 {
		t.Fatalf(
			"old=%+v",
			oldLocation,
		)
	}

	newLocation :=
		chain.TransactionLocationSnapshot(
			hex.EncodeToString(newID),
		)

	if !newLocation.InBlock ||
		newLocation.ConfirmedHeight != 3 {
		t.Fatalf(
			"new=%+v",
			newLocation,
		)
	}
}

func TestTransactionLocationSnapshotConcurrentMutation(
	t *testing.T,
) {
	chain := &Blockchain{
		Blocks:           []*Block{},
		pendingTxs:       []*Transaction{},
		confirmedTxIndex: make(map[string]int),
		pendingTxIndex:   make(map[string]struct{}),
	}

	const iterations = 500

	var wait sync.WaitGroup
	wait.Add(2)

	go func() {
		defer wait.Done()

		for index := 0; index < iterations; index++ {
			txID := []byte{
				byte(index >> 8),
				byte(index),
			}

			tx := &Transaction{
				ID: txID,
			}

			block := &Block{
				Index: index,
				Transactions: []*Transaction{
					tx,
				},
			}

			chain.stateMu.Lock()

			chain.Blocks = append(
				chain.Blocks,
				block,
			)

			chain.indexConfirmedBlockTransactionsLocked(
				block,
			)

			chain.pendingTxs = append(
				chain.pendingTxs,
				tx,
			)

			chain.indexPendingTransactionLocked(
				tx,
			)

			chain.stateMu.Unlock()
		}
	}()

	go func() {
		defer wait.Done()

		for index := 0; index < iterations; index++ {
			txID := []byte{
				byte(index >> 8),
				byte(index),
			}

			_ = chain.TransactionLocationSnapshot(
				hex.EncodeToString(txID),
			)
		}
	}()

	wait.Wait()
}

func TestTransactionLocationIndexMutationHooksPresent(
	t *testing.T,
) {
	blockchainSource, err := os.ReadFile("blockchain.go")
	if err != nil {
		t.Fatalf(
			"read blockchain.go: %v",
			err,
		)
	}

	source := string(blockchainSource)

	required := map[string]int{
		"bc.indexConfirmedBlockTransactionsLocked(nb)":  2,
		"bc.indexConfirmedBlockTransactionsLocked(blk)": 1,
		"bc.indexPendingTransactionLocked(tx)":          1,
		"bc.rebuildTransactionLocationIndexesLocked()":  2,
		"bc.clearPendingTransactionIndexLocked()":       1,
	}

	for fragment, minimum := range required {
		count := strings.Count(
			source,
			fragment,
		)

		if count < minimum {
			t.Fatalf(
				"%q count=%d want>=%d",
				fragment,
				count,
				minimum,
			)
		}
	}

	if !strings.Contains(
		source,
		"blockchain.rebuildTransactionLocationIndexesLocked()",
	) {
		t.Fatal("constructor index rebuild hook missing")
	}

	mutationSource, err :=
		os.ReadFile("state_mutation.go")

	if err != nil {
		t.Fatalf(
			"read state_mutation.go: %v",
			err,
		)
	}

	if !strings.Contains(
		string(mutationSource),
		"bc.indexConfirmedBlockTransactionsLocked(candidate)",
	) {
		t.Fatal("persisted commit index hook missing")
	}
}
