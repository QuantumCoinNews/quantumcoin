package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"quantumcoin/wallet"
)

func nftOwnershipExpectedEntryV1(t *testing.T, chain *Blockchain, record NFTMintRecordV1) NFTOwnershipV1 {
	t.Helper()
	id, err := NFTTokenIDV1(record)
	if err != nil {
		t.Fatal(err)
	}
	return NFTOwnershipV1{TokenID: id, Record: record, MintHeight: 1, MintBlockHash: chain.AllBlocksSnapshot()[1].Hash}
}

func nftOwnershipCheckV1(t *testing.T, chain *Blockchain, want NFTOwnershipV1) NFTOwnershipSnapshotV1 {
	t.Helper()
	snapshot, err := chain.NFTOwnershipSnapshotSynchronizedV1()
	if err != nil {
		t.Fatal(err)
	}
	blocks := chain.AllBlocksSnapshot()
	if snapshot.Height != len(blocks)-1 || !bytes.Equal(snapshot.TipHash, blocks[len(blocks)-1].Hash) || len(snapshot.NFTs) != 1 || !reflect.DeepEqual(snapshot.NFTs[0], want) {
		t.Fatalf("canonical ownership snapshot mismatch: got=%+v want=%+v", snapshot, want)
	}
	entry, found, err := chain.NFTByTokenIDSynchronizedV1(want.TokenID)
	if err != nil || !found || !reflect.DeepEqual(entry, want) {
		t.Fatalf("canonical token lookup: found=%v err=%v entry=%+v", found, err, entry)
	}
	owned, err := chain.NFTsByOwnerSynchronizedV1(want.Record.RecipientPubKeyHash)
	if err != nil || len(owned) != 1 || !reflect.DeepEqual(owned[0], want) {
		t.Fatalf("canonical owner lookup: %v %v", owned, err)
	}
	return snapshot
}

// Construct an approved alternative branch through the actual peer/mining
// entries. Keys and program dates come only from existing ephemeral fixtures.
func nftOwnershipAlternativeV1(t *testing.T, trusted *Blockchain, record NFTMintRecordV1, keys []*ecdsa.PrivateKey) *Blockchain {
	t.Helper()
	branch := DeserializeBlockchain(SerializeBlockchain(trusted))
	nftLiveAdmissionRebindV1(t, branch, trusted)
	branch.SetCoinbaseMaturity(2)
	award := nftLiveAdmissionBlockV1(t, branch, record, keys, 0)
	if err := branch.AddBlockFromPeerSynchronizedV2(award); err != nil {
		t.Fatal(err)
	}
	if _, err := branch.MineBlock(wallet.NewWallet().GetAddress(), testExpectedNextDifficultyV1(t, branch)); err != nil {
		t.Fatal(err)
	}
	return branch
}

func TestNFTOwnershipCanonicalQueriesAreDetachedV1(t *testing.T) {
	_, producer, record, _ := nftTrustedLoadFixtureV1(t)
	want := nftOwnershipExpectedEntryV1(t, producer, record)
	snapshot := nftOwnershipCheckV1(t, producer, want)
	snapshot.TipHash[0] ^= 1
	snapshot.NFTs[0].MintBlockHash[0] ^= 1
	snapshot.NFTs[0].Record.RecipientPubKeyHash[0] ^= 1
	entry, found, err := producer.NFTByTokenIDSynchronizedV1(want.TokenID)
	if err != nil || !found {
		t.Fatalf("token query failed: %v", err)
	}
	entry.Record.RecipientPubKeyHash[0] ^= 1
	entry.MintBlockHash[0] ^= 1
	owner := cloneBytesV1(record.RecipientPubKeyHash)
	owned, err := producer.NFTsByOwnerSynchronizedV1(owner)
	if err != nil || len(owned) != 1 {
		t.Fatalf("owner query failed: %v", err)
	}
	owner[0] ^= 1
	owned[0].Record.RecipientPubKeyHash[0] ^= 1
	owned[0].MintBlockHash[0] ^= 1
	nftOwnershipCheckV1(t, producer, want)
	if _, found, err := producer.NFTByTokenIDSynchronizedV1([32]byte{}); err != nil || found {
		t.Fatalf("unknown token lookup: %v %v", found, err)
	}
	if owned, err := producer.NFTsByOwnerSynchronizedV1(bytes.Repeat([]byte{0xee}, 20)); err != nil || len(owned) != 0 {
		t.Fatalf("unknown owner lookup: %v %v", owned, err)
	}
	if _, err := producer.NFTsByOwnerSynchronizedV1([]byte("display-address")); err == nil {
		t.Fatal("malformed owner hash accepted")
	}
	var missing *Blockchain
	if _, err := missing.NFTOwnershipSnapshotSynchronizedV1(); err == nil {
		t.Fatal("nil chain supplied ownership")
	}
	if _, err := (&Blockchain{}).NFTOwnershipSnapshotSynchronizedV1(); err == nil {
		t.Fatal("empty chain supplied ownership")
	}
	legacy := NewBlockchain(1, 0)
	if empty, err := legacy.NFTOwnershipSnapshotSynchronizedV1(); err != nil || empty.Height != 0 || len(empty.NFTs) != 0 {
		t.Fatalf("legacy chain acquired an NFT: %+v %v", empty, err)
	}
}

func TestNFTOwnershipReplacementAndRestartFollowCanonicalBranchV1(t *testing.T) {
	for _, name := range []string{"recipient_only", "different_token"} {
		t.Run(name, func(t *testing.T) {
			trusted, local, record, keys := nftTrustedLoadFixtureV1(t)
			oldEntry := nftOwnershipExpectedEntryV1(t, local, record)
			oldView := nftOwnershipCheckV1(t, local, oldEntry)
			replacement := record
			replacement.RecipientPubKeyHash = bytes.Repeat([]byte{0x77}, 20)
			if name == "different_token" {
				replacement.MetadataHash[0] ^= 1
			}
			branch := nftOwnershipAlternativeV1(t, trusted, replacement, keys)
			want := nftOwnershipExpectedEntryV1(t, branch, replacement)
			incoming := branch.AllBlocksSnapshot()
			if err := local.ReplaceChainFromPeerSynchronizedV1(incoming); err != nil {
				t.Fatal(err)
			}
			incoming[1].NFTMintRecordsV1[0].RecipientPubKeyHash[0] ^= 1
			view := nftOwnershipCheckV1(t, local, want)
			if owned, err := local.NFTsByOwnerSynchronizedV1(record.RecipientPubKeyHash); err != nil || len(owned) != 0 {
				t.Fatalf("abandoned branch retained ownership: %v %v", owned, err)
			}
			if name == "recipient_only" && oldEntry.TokenID != want.TokenID {
				t.Fatal("recipient-only branch change altered canonical token identity")
			}
			if name == "different_token" {
				if oldEntry.TokenID == want.TokenID {
					t.Fatal("changed metadata retained the old token identity")
				}
				if _, found, err := local.NFTByTokenIDSynchronizedV1(oldEntry.TokenID); err != nil || found {
					t.Fatalf("abandoned token survived canonical reorg: %v %v", found, err)
				}
			}
			if oldView.Height != 1 || !reflect.DeepEqual(oldView.NFTs[0], oldEntry) {
				t.Fatal("reorg mutated a caller's detached previous view")
			}
			path := nftTrustedLoadWriteV1(t, local)
			restarted, err := LoadBlockchainFromFileWithLocalContextV1(path, trusted)
			if err != nil {
				t.Fatal(err)
			}
			restored := nftOwnershipCheckV1(t, restarted, want)
			if !reflect.DeepEqual(view, restored) {
				t.Fatal("restart restored stale or incomplete ownership")
			}
		})
	}
}

func TestNFTOwnershipFailedCommitAndRejectedChainLeaveStateUnchangedV1(t *testing.T) {
	trusted, _, record, keys := nftTrustedLoadFixtureV1(t)
	before, err := trusted.NFTOwnershipSnapshotSynchronizedV1()
	if err != nil {
		t.Fatal(err)
	}
	award := nftLiveAdmissionBlockV1(t, trusted, record, keys, 0)
	badPath := filepath.Join(t.TempDir(), "missing-parent", "chain.dat")
	if err := trusted.CommitValidatedBlockAndPersist(award, badPath); err == nil || !strings.Contains(err.Error(), "persist committed block") {
		t.Fatalf("fixture did not reach persistence failure after validation: %v", err)
	}
	after, err := trusted.NFTOwnershipSnapshotSynchronizedV1()
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("failed persistence leaked NFT ownership: %+v %v", after, err)
	}
	path := filepath.Join(t.TempDir(), "chain.dat")
	if err := trusted.CommitValidatedBlockAndPersist(award, path); err != nil {
		t.Fatal(err)
	}
	want := nftOwnershipExpectedEntryV1(t, trusted, record)
	committed := nftOwnershipCheckV1(t, trusted, want)
	loaded, err := LoadBlockchainFromFileWithLocalContextV1(path, trusted)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(committed, nftOwnershipCheckV1(t, loaded, want)) {
		t.Fatal("successful persisted commit lost ownership on load")
	}
	second := record
	second.RecipientPubKeyHash = bytes.Repeat([]byte{0x88}, 20)
	duplicate := nftLiveAdmissionBlockV1(t, trusted, second, keys, 0)
	if err := trusted.AddBlockFromPeerSynchronizedV2(duplicate); err == nil || !strings.Contains(err.Error(), "duplicate NFT award period") {
		t.Fatalf("second owner was not rejected at the period gate: %v", err)
	}
	genesis := DeserializeBlockchain(SerializeBlockchain(trusted))
	genesis.Blocks = genesis.Blocks[:1]
	nftLiveAdmissionRebindV1(t, genesis, trusted)
	genesis.UpdateUTXOSetSynchronized()
	branch := nftOwnershipAlternativeV1(t, genesis, second, keys)
	poisoned := branch.AllBlocksSnapshot()
	poisoned[1].NFTMintRecordsV1[0].RecipientPubKeyHash[0] ^= 1
	if err := trusted.ReplaceChainFromPeerSynchronizedV1(poisoned); err == nil {
		t.Fatal("replacement with changed NFT evidence accepted")
	}
	current, err := trusted.NFTOwnershipSnapshotSynchronizedV1()
	if err != nil || !reflect.DeepEqual(current, committed) {
		t.Fatalf("rejected admission/replacement changed owner: %+v %v", current, err)
	}
}

func TestNFTOwnershipRejectsUntrustedAndTamperedEvidenceV1(t *testing.T) {
	trusted, producer, record, keys := nftTrustedLoadFixtureV1(t)
	for _, name := range []string{"raw_decode", "changed_recipient", "missing_proof", "orphan_proof", "unknown_signers", "genesis_award", "duplicate_period", "nil_block"} {
		t.Run(name, func(t *testing.T) {
			wire := DeserializeBlockchain(SerializeBlockchain(producer))
			if name != "raw_decode" {
				if err := wire.BindPeerValidationContextFromLocalSynchronizedV1(trusted); err != nil {
					t.Fatal(err)
				}
			}
			switch name {
			case "changed_recipient":
				wire.Blocks[1].NFTMintRecordsV1[0].RecipientPubKeyHash[0] ^= 1
			case "missing_proof":
				wire.Blocks[1].NFTAwardProofsV1 = nil
			case "orphan_proof":
				wire.Blocks[1].NFTMintRecordsV1 = nil
			case "unknown_signers":
				_, _, rogue := nftLiveAdmissionFixtureV1(t, 1)
				wire.Blocks[1] = nftLiveAdmissionBlockV1(t, trusted, record, rogue, 0)
			case "genesis_award":
				wire.Blocks[0].NFTMintRecordsV1 = []NFTMintRecordV1{record}
			case "duplicate_period":
				wire.Blocks = append(wire.Blocks, nftLiveAdmissionBlockV1(t, producer, record, keys, 0))
			case "nil_block":
				wire.Blocks[1] = nil
			}
			if snapshot, err := wire.NFTOwnershipSnapshotSynchronizedV1(); err == nil || len(snapshot.NFTs) != 0 {
				t.Fatalf("untrusted evidence supplied complete/partial ownership: %+v %v", snapshot, err)
			}
		})
	}
	nftOwnershipCheckV1(t, producer, nftOwnershipExpectedEntryV1(t, producer, record))
}

func TestNFTOwnershipConcurrentReadersObserveCompleteReplacementV1(t *testing.T) {
	trusted, local, record, keys := nftTrustedLoadFixtureV1(t)
	oldView := nftOwnershipCheckV1(t, local, nftOwnershipExpectedEntryV1(t, local, record))
	replacement := record
	replacement.RecipientPubKeyHash = bytes.Repeat([]byte{0x99}, 20)
	branch := nftOwnershipAlternativeV1(t, trusted, replacement, keys)
	newView := nftOwnershipCheckV1(t, branch, nftOwnershipExpectedEntryV1(t, branch, replacement))
	start := make(chan struct{})
	problems := make(chan error, 4)
	var ready, workers sync.WaitGroup
	ready.Add(4)
	for i := 0; i < 4; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			ready.Done()
			<-start
			for j := 0; j < 8; j++ {
				view, err := local.NFTOwnershipSnapshotSynchronizedV1()
				if err != nil || (!reflect.DeepEqual(view, oldView) && !reflect.DeepEqual(view, newView)) {
					problems <- fmt.Errorf("reader observed mixed branch/tip/owner: %+v %v", view, err)
					return
				}
			}
		}()
	}
	ready.Wait()
	close(start)
	replaceErr := local.ReplaceChainFromPeerSynchronizedV1(branch.AllBlocksSnapshot())
	workers.Wait()
	close(problems)
	if replaceErr != nil {
		t.Fatal(replaceErr)
	}
	for err := range problems {
		t.Error(err)
	}
	if view := nftOwnershipCheckV1(t, local, nftOwnershipExpectedEntryV1(t, branch, replacement)); !reflect.DeepEqual(view, newView) {
		t.Fatal("final query did not follow the replacement tip")
	}
}
