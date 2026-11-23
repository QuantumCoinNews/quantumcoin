package blockchain

import (
	"encoding/hex"
	"path/filepath"
	"sync"
	"testing"

	"quantumcoin/wallet"
)

func assertQueryIndexMatchesLedgerV1(t *testing.T, chain *Blockchain, ids ...[]byte) {
	t.Helper()
	for _, id := range ids {
		want := TransactionLocation{ConfirmedHeight: -1}
		key := hex.EncodeToString(id)
		chain.stateMu.RLock()
		for _, block := range chain.Blocks {
			if block == nil {
				continue
			}
			for _, tx := range block.Transactions {
				if tx != nil && hex.EncodeToString(tx.ID) == key && !want.InBlock {
					want.InBlock = true
					want.ConfirmedHeight = block.Index
				}
			}
		}
		for _, tx := range chain.pendingTxs {
			if tx != nil && hex.EncodeToString(tx.ID) == key {
				want.InMempool = true
			}
		}
		chain.stateMu.RUnlock()
		for range 16 {
			if got := chain.TransactionLocationSnapshot(key); got != want {
				t.Fatalf("index differs from authoritative ledger: got=%+v want=%+v", got, want)
			}
		}
	}
}

func TestQueryIndexPendingToConfirmedOnCanonicalReplacement(t *testing.T) {
	chain, _, _, spend := newFeeConsensusTestStateV1(t)
	assertQueryIndexMatchesLedgerV1(t, chain, spend.ID)
	if err := chain.AddTransaction(spend); err != nil {
		t.Fatal(err)
	}
	if !chain.TransactionLocationSnapshot(hex.EncodeToString(spend.ID)).InMempool || chain.PendingTransactionCountSnapshot() != 1 {
		t.Fatal("admission not indexed")
	}
	assertQueryIndexMatchesLedgerV1(t, chain, spend.ID)
	miner := wallet.NewWallet().GetAddress()
	genesis := chain.GetLastBlock()
	block, err := NewBlock(1, []*Transaction{feeTestCoinbaseV1(t, miner, 53), spend}, genesis.Hash, miner, testExpectedNextDifficultyV1(t, chain))
	if err != nil {
		t.Fatal(err)
	}
	if err := chain.ReplaceChain([]*Block{genesis, block}); err != nil {
		t.Fatal(err)
	}
	assertQueryIndexMatchesLedgerV1(t, chain, spend.ID, block.Transactions[0].ID)
	location := chain.TransactionLocationSnapshot(hex.EncodeToString(spend.ID))
	if !location.InBlock || location.InMempool || location.ConfirmedHeight != 1 || chain.PendingTransactionCountSnapshot() != 0 {
		t.Fatal("replacement left stale index or mempool", location)
	}
}

func TestQueryIndexMiningAndDiskReload(t *testing.T) {
	t.Setenv("QC_DIFFICULTY_BITS", "8")
	chain := NewIsolatedLocalBlockchainV1(50, 0)
	block, err := chain.MineBlock(wallet.NewWallet().GetAddress(), 8)
	if err != nil {
		t.Fatal(err)
	}
	id := block.Transactions[0].ID
	assertQueryIndexMatchesLedgerV1(t, chain, id)
	file := filepath.Join(t.TempDir(), "query-chain.dat")
	if err := chain.SaveToFile(file); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadBlockchainFromFile(file)
	if err != nil {
		t.Fatal(err)
	}
	assertQueryIndexMatchesLedgerV1(t, loaded, id, make([]byte, 32))
	location := loaded.TransactionLocationSnapshot(hex.EncodeToString(id))
	if !location.InBlock || location.ConfirmedHeight != block.Index {
		t.Fatal("loaded index lost mined transaction", location)
	}
	if found := loaded.BlockByHashSnapshot(block.Hash); found == nil || found.Index != block.Index {
		t.Fatal("loaded block hash index lost mined block")
	}
	fork := NewIsolatedLocalBlockchainV1(50, 0)
	for range 2 {
		// Legacy coinbase IDs commit seconds and outputs; distinct test
		// recipients avoid duplicate IDs when several blocks finish in a second.
		if _, err := fork.MineBlock(wallet.NewWallet().GetAddress(), 8); err != nil {
			t.Fatal(err)
		}
	}
	if err := loaded.ReplaceChain(fork.BlocksSnapshot()); err != nil {
		t.Fatal(err)
	}
	if loaded.BlockByHashSnapshot(block.Hash) != nil {
		t.Fatal("reorg retained orphan block in hash index")
	}
	assertQueryIndexMatchesLedgerV1(t, loaded, id)
	for _, source := range fork.BlocksSnapshot() {
		found := loaded.BlockByHashSnapshot(source.Hash)
		if found == nil || found.Index != source.Index {
			t.Fatal("replacement block absent from hash index")
		}
	}
}

func TestQueryBlockHashIndexColdReadAndDefensiveOwnership(t *testing.T) {
	chain := &Blockchain{Blocks: []*Block{nil, {Index: 1, Hash: []byte{1, 2}, Metadata: map[string]string{"owner": "original"}}, {Index: 2, Hash: []byte{1, 2}}}}
	for range 2 {
		block := chain.BlockByHashSnapshot([]byte{1, 2})
		if block == nil || block.Index != 1 {
			t.Fatal("cold/warm first-match lookup changed")
		}
		block.Hash[0] = 9
		block.Metadata["owner"] = "caller"
		if chain.Blocks[1].Hash[0] != 1 || chain.Blocks[1].Metadata["owner"] != "original" {
			t.Fatal("query exposes owned block state")
		}
	}
	if chain.BlockByHashSnapshot([]byte{9, 9}) != nil {
		t.Fatal("unknown hash found")
	}
}

func TestQueryBlockHashIndexOwnsCanonicalAppend(t *testing.T) {
	t.Setenv("QC_DIFFICULTY_BITS", "8")
	chain := NewIsolatedLocalBlockchainV1(50, 0)
	miner := wallet.NewWallet().GetAddress()
	block := chain.AddBlock([]*Transaction{feeTestCoinbaseV1(t, miner, 50)}, miner, 8)
	if block == nil {
		t.Fatal("append failed")
	}
	hash := append([]byte(nil), block.Hash...)
	block.Hash[0] ^= 0xff
	block.Metadata["external-candidate"] = "changed"
	canonical := chain.BlockByHashSnapshot(hash)
	if canonical == nil || hex.EncodeToString(canonical.Hash) != hex.EncodeToString(hash) || canonical.Metadata["external-candidate"] != "" {
		t.Fatal("index points to caller-owned candidate")
	}
}

func TestQueryBlockHashIndexConcurrentCanonicalAppend(t *testing.T) {
	t.Setenv("QC_DIFFICULTY_BITS", "8")
	chain := NewIsolatedLocalBlockchainV1(50, 0)
	genesis := chain.BlockByIndexSnapshot(0)
	stop := make(chan struct{})
	failures := make(chan string, 1)
	var readers sync.WaitGroup
	readers.Go(func() {
		for {
			select {
			case <-stop:
				return
			default:
			}
			if block := chain.BlockByHashSnapshot(genesis.Hash); block == nil || block.Index != 0 {
				select {
				case failures <- "concurrent lookup lost genesis":
				default:
				}
				return
			}
			chain.TransactionLocationSnapshot(hex.EncodeToString(genesis.Transactions[0].ID))
		}
	})
	defer func() {
		close(stop)
		readers.Wait()
		select {
		case failure := <-failures:
			t.Error(failure)
		default:
		}
	}()
	for range 3 {
		block, err := chain.MineBlock(wallet.NewWallet().GetAddress(), 8)
		if err != nil {
			t.Fatal(err)
		}
		if found := chain.BlockByHashSnapshot(block.Hash); found == nil || found.Index != block.Index {
			t.Fatal("append not indexed")
		}
	}
}
