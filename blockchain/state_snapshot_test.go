package blockchain

import "testing"

func newStateSnapshotTestChain() *Blockchain {
	confirmed := &Transaction{
		ID:     []byte{0x01},
		Sender: "confirmed-sender",
		Outputs: []TransactionOutput{
			{
				Amount:     7,
				PubKeyHash: []byte{0x11, 0x12},
			},
		},
	}

	pending := &Transaction{
		ID:     []byte{0x02},
		Sender: "pending-sender",
		Outputs: []TransactionOutput{
			{
				Amount:     3,
				PubKeyHash: []byte{0x21, 0x22},
			},
		},
	}

	return &Blockchain{
		Blocks: []*Block{
			{
				Index:    0,
				Hash:     []byte{0x31},
				PrevHash: []byte{0x30},
				Transactions: []*Transaction{
					confirmed,
				},
				Metadata: map[string]string{
					"source": "state-snapshot-test",
				},
			},
		},
		UTXO: map[string][]TransactionOutput{
			"01": {
				{
					Amount:     7,
					PubKeyHash: []byte{0x41, 0x42},
				},
			},
		},
		TotalSupply:      25_500_000,
		coinbaseMaturity: 5,
		pendingTxs: []*Transaction{
			pending,
		},
	}
}

func TestBlockchainStateSnapshotIsDefensive(
	t *testing.T,
) {
	chain := newStateSnapshotTestChain()

	snapshot := chain.StateSnapshot()

	if len(snapshot.Blocks) != 1 {
		t.Fatalf(
			"blocks=%d want=1",
			len(snapshot.Blocks),
		)
	}

	if len(snapshot.PendingTransactions) != 1 {
		t.Fatalf(
			"pending=%d want=1",
			len(snapshot.PendingTransactions),
		)
	}

	snapshot.Blocks[0].Hash[0] = 0xff
	snapshot.Blocks[0].Metadata["source"] = "attacker"
	snapshot.Blocks[0].
		Transactions[0].
		Outputs[0].
		PubKeyHash[0] = 0xff

	snapshot.UTXO["01"][0].
		PubKeyHash[0] = 0xff

	snapshot.PendingTransactions[0].
		Outputs[0].
		PubKeyHash[0] = 0xff

	if chain.Blocks[0].Hash[0] != 0x31 {
		t.Fatal(
			"snapshot mutated live block hash",
		)
	}

	if chain.Blocks[0].Metadata["source"] !=
		"state-snapshot-test" {
		t.Fatal(
			"snapshot mutated live metadata",
		)
	}

	if chain.Blocks[0].
		Transactions[0].
		Outputs[0].
		PubKeyHash[0] != 0x11 {
		t.Fatal(
			"snapshot mutated confirmed transaction",
		)
	}

	if chain.UTXO["01"][0].
		PubKeyHash[0] != 0x41 {
		t.Fatal(
			"snapshot mutated live UTXO",
		)
	}

	if chain.pendingTxs[0].
		Outputs[0].
		PubKeyHash[0] != 0x21 {
		t.Fatal(
			"snapshot mutated live pending transaction",
		)
	}
}

func TestBlocksRangeSnapshotIsBoundedAndDefensive(
	t *testing.T,
) {
	chain := newStateSnapshotTestChain()

	chain.Blocks = append(
		chain.Blocks,
		&Block{
			Index: 1,
			Hash:  []byte{0x32},
		},
		&Block{
			Index: 2,
			Hash:  []byte{0x33},
		},
	)

	blocks, total :=
		chain.BlocksRangeSnapshot(
			1,
			1,
		)

	if total != 3 {
		t.Fatalf(
			"total=%d want=3",
			total,
		)
	}

	if len(blocks) != 1 {
		t.Fatalf(
			"len=%d want=1",
			len(blocks),
		)
	}

	if blocks[0].Index != 1 {
		t.Fatalf(
			"index=%d want=1",
			blocks[0].Index,
		)
	}

	blocks[0].Hash[0] = 0xff

	if chain.Blocks[1].Hash[0] != 0x32 {
		t.Fatal(
			"bounded block snapshot aliased live state",
		)
	}
}

func TestPendingTransactionsRangeSnapshotIsBoundedAndDefensive(
	t *testing.T,
) {
	chain := newStateSnapshotTestChain()

	chain.pendingTxs = append(
		chain.pendingTxs,
		&Transaction{
			ID: []byte{0x03},
			Outputs: []TransactionOutput{
				{
					Amount:     4,
					PubKeyHash: []byte{0x51},
				},
			},
		},
	)

	pending, total :=
		chain.PendingTransactionsRangeSnapshot(
			0,
			1,
		)

	if total != 2 {
		t.Fatalf(
			"total=%d want=2",
			total,
		)
	}

	if len(pending) != 1 {
		t.Fatalf(
			"len=%d want=1",
			len(pending),
		)
	}

	pending[0].
		Outputs[0].
		PubKeyHash[0] = 0xff

	if chain.pendingTxs[0].
		Outputs[0].
		PubKeyHash[0] != 0x21 {
		t.Fatal(
			"bounded mempool snapshot aliased live state",
		)
	}
}

func TestChainSummarySnapshotUsesDetachedTip(
	t *testing.T,
) {
	chain := newStateSnapshotTestChain()

	summary := chain.ChainSummarySnapshot()

	if summary.Height != 0 {
		t.Fatalf(
			"height=%d want=0",
			summary.Height,
		)
	}

	if summary.BlockCount != 1 {
		t.Fatalf(
			"blockCount=%d want=1",
			summary.BlockCount,
		)
	}

	if summary.TotalSupply != 25_500_000 {
		t.Fatalf(
			"totalSupply=%d",
			summary.TotalSupply,
		)
	}

	if summary.Tip == nil {
		t.Fatal("tip is nil")
	}

	summary.Tip.Hash[0] = 0xff

	if chain.Blocks[0].Hash[0] != 0x31 {
		t.Fatal(
			"summary tip aliased live state",
		)
	}
}

func TestPendingTransactionCountSnapshotV1(t *testing.T) {
	chain := &Blockchain{
		pendingTxs: make([]*Transaction, 2),
	}

	if got := chain.PendingTransactionCountSnapshot(); got != 2 {
		t.Fatalf("pending transaction count mismatch: got=%d want=2", got)
	}

	var nilChain *Blockchain
	if got := nilChain.PendingTransactionCountSnapshot(); got != 0 {
		t.Fatalf("nil blockchain pending count mismatch: got=%d want=0", got)
	}
}
