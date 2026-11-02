package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	qconsensus "quantumcoin/consensus"
	"quantumcoin/wallet"
)

// This interface also lets the diagnostic compile before the entry exists.
type nftDurableMiningEntryForTestV1 interface {
	MineBlockWithNFTAwardAndPersistSynchronizedV1(string, NFTMintRecordV1, NFTAwardProofV1, string) (*Block, error)
}

func nftDurableMiningEntryV1(t *testing.T, bc *Blockchain) nftDurableMiningEntryForTestV1 {
	t.Helper()
	entry, ok := any(bc).(nftDurableMiningEntryForTestV1)
	if !ok {
		t.Fatal("QMA0066_DURABLE_RED: approved NFT mining has no disk-before-RAM entry")
	}
	return entry
}

type nftDurableMiningFixtureDataV1 struct {
	chain    *Blockchain
	filename string
	record   NFTMintRecordV1
	proof    NFTAwardProofV1
	spend    *Transaction
	keys     []*ecdsa.PrivateKey
}

func nftDurableMiningFixtureV1(t *testing.T) *nftDurableMiningFixtureDataV1 {
	t.Helper()
	chain, policy, keys := nftLiveAdmissionFixtureV1(t, 2)
	chain.SetCoinbaseMaturity(0)
	sender, receiver := wallet.NewWallet(), wallet.NewWallet()
	funding, err := chain.MineBlock(sender.GetAddress(), testExpectedNextDifficultyV1(t, chain))
	if err != nil {
		t.Fatal(err)
	}
	tx := funding.Transactions[0]
	units, err := qconsensus.WholeQCAmountToProtocolUnitsV1(qconsensus.ProtocolNFTV4, int64(tx.Outputs[0].Amount))
	if err != nil {
		t.Fatal(err)
	}
	spend := &Transaction{
		Inputs:    []TransactionInput{{TxID: cloneBytesV1(tx.ID), OutIndex: 0}},
		Outputs:   []TransactionOutput{{Amount: int(units) - 1, PubKeyHash: wallet.HashPubKey(receiver.PublicKey)}},
		Timestamp: time.Now().UTC(), Sender: sender.GetAddress(), Amount: float64(units - 1),
	}
	spend.ID = spend.Hash()
	if err := spend.Sign(sender.PrivateKey); err != nil {
		t.Fatal(err)
	}
	if err := chain.AddTransaction(spend); err != nil {
		t.Fatal(err)
	}
	record := nftMonthlyPolicyRecordV1(policy, 0)
	proof := nftMonthlyPolicyBlockV1(t, chain, []NFTMintRecordV1{record}, keys).NFTAwardProofsV1[0]
	filename := filepath.Join(t.TempDir(), "chain.dat")
	if err := chain.SaveToFile(filename); err != nil {
		t.Fatal(err)
	}
	return &nftDurableMiningFixtureDataV1{chain: chain, filename: filename, record: record, proof: proof, spend: spend, keys: keys}
}

func nftDurableReadFileV1(t *testing.T, filename string) []byte {
	t.Helper()
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestNFTDurableMiningPersistsAwardFeeAndDetachedOwnershipV1(t *testing.T) {
	entry := nftDurableMiningEntryV1(t, (*Blockchain)(nil))
	// The exact missing-entry diagnostic runs before fixture work.
	f := nftDurableMiningFixtureV1(t)
	entry = nftDurableMiningEntryV1(t, f.chain)
	before, subsidy := f.chain.TotalMinted(), f.chain.ExpectedSubsidyForHeightV1(2)
	subsidyUnits, err := f.chain.ExpectedSubsidyUnitsForHeightV1(2)
	if err != nil {
		t.Fatal(err)
	}
	block, err := entry.MineBlockWithNFTAwardAndPersistSynchronizedV1(wallet.NewWallet().GetAddress(), f.record, f.proof, f.filename)
	if err != nil {
		t.Fatal(err)
	}
	_, paid, err := blockCoinbaseTotalV1(block, false)
	if err != nil || paid != subsidyUnits+1 || f.chain.TotalMinted() != before+subsidy {
		t.Fatalf("persistent mint changed canonical subsidy/one-atom fee: paid=%d err=%v", paid, err)
	}
	location := f.chain.TransactionLocationSnapshot(hex.EncodeToString(f.spend.ID))
	if !location.InBlock || location.InMempool || location.ConfirmedHeight != 2 || len(f.chain.StateSnapshot().PendingTransactions) != 0 {
		t.Fatal("durable success did not confirm the spend and clear its mempool index")
	}
	loaded, err := LoadBlockchainFromFileWithLocalContextV1(f.filename, f.chain)
	if err != nil {
		t.Fatal(err)
	}
	owned, err := f.chain.NFTOwnershipSnapshotSynchronizedV1()
	if err != nil || len(owned.NFTs) != 1 || owned.Height != 2 {
		t.Fatalf("canonical ownership unavailable after durable success: %+v %v", owned, err)
	}
	restored, err := loaded.NFTOwnershipSnapshotSynchronizedV1()
	if err != nil || !reflect.DeepEqual(owned, restored) || !reflect.DeepEqual(f.chain.StateSnapshot().UTXO, loaded.StateSnapshot().UTXO) {
		t.Fatalf("persisted block/UTXO/ownership did not survive trusted reload: %v", err)
	}
	if _, err := LoadBlockchainFromFile(f.filename); err == nil {
		t.Fatal("durable file supplied its own NFT trust")
	}
	fileBefore := nftDurableReadFileV1(t, f.filename)
	stateBefore := f.chain.StateSnapshot()
	if _, err := entry.MineBlockWithNFTAwardAndPersistSynchronizedV1(wallet.NewWallet().GetAddress(), f.record, f.proof, f.filename); err == nil {
		t.Fatal("duplicate month minted again")
	}
	if !reflect.DeepEqual(stateBefore, f.chain.StateSnapshot()) || !bytes.Equal(fileBefore, nftDurableReadFileV1(t, f.filename)) {
		t.Fatal("duplicate retry changed RAM or persisted bytes")
	}
	block.Hash[0] ^= 1
	block.NFTMintRecordsV1[0].RecipientPubKeyHash[0] ^= 1
	block.NFTAwardProofsV1[0].Approvals[0].Signature[0] ^= 1
	f.record.RecipientPubKeyHash[0] ^= 1
	f.proof.Approvals[0].PublicKey[0] ^= 1
	if !reflect.DeepEqual(stateBefore, f.chain.StateSnapshot()) || !bytes.Equal(fileBefore, nftDurableReadFileV1(t, f.filename)) {
		t.Fatal("returned block or submitted evidence retained live state")
	}
}

func TestNFTDurableMiningPreservesStateOnWriteAndInstallFailureV1(t *testing.T) {
	for _, name := range []string{"temporary_file", "install", "empty_filename"} {
		t.Run(name, func(t *testing.T) {
			f := nftDurableMiningFixtureV1(t)
			entry := nftDurableMiningEntryV1(t, f.chain)
			parent := t.TempDir()
			target := ""
			var sentinel string
			if name == "temporary_file" {
				sentinel = filepath.Join(parent, "parent-is-file")
				if err := os.WriteFile(sentinel, []byte("keep-parent"), 0600); err != nil {
					t.Fatal(err)
				}
				target = filepath.Join(sentinel, "chain.dat")
			} else if name == "install" {
				target = filepath.Join(parent, "target-is-directory")
				if err := os.Mkdir(target, 0700); err != nil {
					t.Fatal(err)
				}
				sentinel = filepath.Join(target, "keep.txt")
				if err := os.WriteFile(sentinel, []byte("keep-directory"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			var sentinelBefore []byte
			if sentinel != "" {
				sentinelBefore = nftDurableReadFileV1(t, sentinel)
			}
			state := f.chain.StateSnapshot()
			owned, err := f.chain.NFTOwnershipSnapshotSynchronizedV1()
			if err != nil {
				t.Fatal(err)
			}
			file := nftDurableReadFileV1(t, f.filename)
			location := f.chain.TransactionLocationSnapshot(hex.EncodeToString(f.spend.ID))
			_, err = entry.MineBlockWithNFTAwardAndPersistSynchronizedV1(wallet.NewWallet().GetAddress(), f.record, f.proof, target)
			if err == nil || (name != "empty_filename" && !strings.Contains(err.Error(), "persistent NFT mining")) {
				t.Fatalf("expected actual file-boundary failure: %v", err)
			}
			after, ownershipErr := f.chain.NFTOwnershipSnapshotSynchronizedV1()
			if ownershipErr != nil || !reflect.DeepEqual(state, f.chain.StateSnapshot()) || !reflect.DeepEqual(owned, after) ||
				location != f.chain.TransactionLocationSnapshot(hex.EncodeToString(f.spend.ID)) || !bytes.Equal(file, nftDurableReadFileV1(t, f.filename)) {
				t.Fatalf("I/O failure changed blocks, UTXO, mempool, indexes, ownership or disk: %v", ownershipErr)
			}
			if sentinel != "" && !bytes.Equal(sentinelBefore, nftDurableReadFileV1(t, sentinel)) {
				t.Fatal("failed install modified existing target/parent data")
			}
			leftovers, err := filepath.Glob(filepath.Join(parent, ".qma-nft-commit-*"))
			if err != nil || len(leftovers) != 0 {
				t.Fatalf("failed commit retained temporary files: %v %v", leftovers, err)
			}
			if _, err := entry.MineBlockWithNFTAwardAndPersistSynchronizedV1(wallet.NewWallet().GetAddress(), f.record, f.proof, f.filename); err != nil {
				t.Fatalf("failed write consumed the monthly award or pending spend: %v", err)
			}
		})
	}
}

func TestNFTDurableMiningRejectsChangedEvidenceBeforeWritingV1(t *testing.T) {
	f := nftDurableMiningFixtureV1(t)
	state, file := f.chain.StateSnapshot(), nftDurableReadFileV1(t, f.filename)
	record := f.record
	record.MetadataHash[0] ^= 1
	if _, err := nftDurableMiningEntryV1(t, f.chain).MineBlockWithNFTAwardAndPersistSynchronizedV1(wallet.NewWallet().GetAddress(), record, f.proof, f.filename); err == nil {
		t.Fatal("changed metadata accepted with original signatures")
	}
	if !reflect.DeepEqual(state, f.chain.StateSnapshot()) || !bytes.Equal(file, nftDurableReadFileV1(t, f.filename)) {
		t.Fatal("invalid proof changed live or persisted state")
	}
	if temporary, err := filepath.Glob(filepath.Join(filepath.Dir(f.filename), ".qma-nft-commit-*")); err != nil || len(temporary) != 0 {
		t.Fatal("invalid proof wrote a prospective chain file")
	}
}

func TestNFTDurableMiningSamePeriodRacePersistsOnlyWinnerV1(t *testing.T) {
	f := nftDurableMiningFixtureV1(t)
	entry := nftDurableMiningEntryV1(t, f.chain)
	records := []NFTMintRecordV1{f.record, f.record}
	records[1].RecipientPubKeyHash = cloneBytesV1(records[1].RecipientPubKeyHash)
	records[1].RecipientPubKeyHash[0] ^= 1
	proofs := []NFTAwardProofV1{f.proof, nftMonthlyPolicyBlockV1(t, f.chain, records[1:], f.keys).NFTAwardProofsV1[0]}
	type result struct {
		block *Block
		err   error
	}
	start, results := make(chan struct{}), make(chan result, 2)
	miner := wallet.NewWallet().GetAddress()
	var workers sync.WaitGroup
	for i := 0; i < 2; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			<-start
			block, err := entry.MineBlockWithNFTAwardAndPersistSynchronizedV1(miner, records[i], proofs[i], f.filename)
			results <- result{block, err}
		}(i)
	}
	close(start)
	workers.Wait()
	close(results)
	accepted, rejected := 0, 0
	var winner *Block
	for got := range results {
		if got.err == nil {
			accepted++
			winner = got.block
		} else {
			rejected++
		}
	}
	if accepted != 1 || rejected != 1 || f.chain.BestHeightSnapshot() != 2 || winner == nil {
		t.Fatalf("same month not serialized at disk/RAM boundary: accepted=%d rejected=%d", accepted, rejected)
	}
	loaded, err := LoadBlockchainFromFileWithLocalContextV1(f.filename, f.chain)
	if err != nil {
		t.Fatal(err)
	}
	ownership, err := loaded.NFTOwnershipSnapshotSynchronizedV1()
	if err != nil || len(ownership.NFTs) != 1 || ownership.Height != 2 ||
		!bytes.Equal(ownership.TipHash, winner.Hash) || !reflect.DeepEqual(ownership.NFTs[0].Record, winner.NFTMintRecordsV1[0]) {
		t.Fatalf("disk did not retain exactly the accepted award: %+v %v", ownership, err)
	}
	// Gob normalizes empty slices. Compare decoded block contents on both
	// sides; serialized byte order is not a state fingerprint.
	normalized := DeserializeBlockchain(SerializeBlockchain(f.chain))
	if !reflect.DeepEqual(normalized.AllBlocksSnapshot(), loaded.AllBlocksSnapshot()) || !reflect.DeepEqual(f.chain.StateSnapshot().UTXO, loaded.StateSnapshot().UTXO) {
		t.Fatal("concurrent monthly admission separated RAM from disk")
	}
}
