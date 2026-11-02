package node

import (
	"bytes"
	"os"
	"reflect"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/wallet"
)

func TestNFTOwnershipRuntimeReloadRefreshesAndPreservesOnFailureV1(t *testing.T) {
	runtime, producer, record, _ := nftTrustedReloadFixtureV1(t)
	genesis, _ := runtime.Snapshot()
	if empty, err := genesis.NFTOwnershipSnapshotSynchronizedV1(); err != nil || len(empty.NFTs) != 0 {
		t.Fatalf("runtime supplied ownership before reload: %+v %v", empty, err)
	}
	if changed, err := runtime.ReloadChainFromDiskIfChanged(); err != nil || !changed {
		t.Fatalf("actual NFT reload failed: %v %v", changed, err)
	}
	loaded, _ := runtime.Snapshot()
	id, err := blockchain.NFTTokenIDV1(record)
	if err != nil {
		t.Fatal(err)
	}
	entry, found, err := loaded.NFTByTokenIDSynchronizedV1(id)
	if err != nil || !found || !bytes.Equal(entry.Record.RecipientPubKeyHash, record.RecipientPubKeyHash) || !bytes.Equal(entry.MintBlockHash, producer.AllBlocksSnapshot()[1].Hash) {
		t.Fatalf("actual reload lost NFT ownership: %+v %v %v", entry, found, err)
	}
	owned, err := loaded.NFTsByOwnerSynchronizedV1(record.RecipientPubKeyHash)
	if err != nil || len(owned) != 1 {
		t.Fatalf("actual reload owner query failed: %v %v", owned, err)
	}
	// A valid replacement file may omit the old award entirely. Build its
	// alternative history through the existing canonical local mining gate.
	branch := blockchain.DeserializeBlockchain(blockchain.SerializeBlockchain(genesis))
	if err := branch.BindPeerValidationContextFromLocalSynchronizedV1(genesis); err != nil {
		t.Fatal(err)
	}
	branch.SetCoinbaseMaturity(runtime.Config.CoinbaseMaturity)
	for i := 0; i < 2; i++ {
		difficulty, err := branch.ExpectedNextDifficultyV1()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := branch.MineBlock(wallet.NewWallet().GetAddress(), difficulty); err != nil {
			t.Fatal(err)
		}
	}
	if err := branch.SaveToFile(runtime.Config.ChainFile); err != nil {
		t.Fatal(err)
	}
	if changed, err := runtime.ReloadChainFromDiskIfChanged(); err != nil || !changed {
		t.Fatalf("actual alternate-history reload failed: %v %v", changed, err)
	}
	current, _ := runtime.Snapshot()
	before, err := current.NFTOwnershipSnapshotSynchronizedV1()
	if err != nil || before.Height != 2 || len(before.NFTs) != 0 || !bytes.Equal(before.TipHash, branch.AllBlocksSnapshot()[2].Hash) {
		t.Fatalf("runtime kept abandoned branch ownership: %+v %v", before, err)
	}
	if _, found, err := current.NFTByTokenIDSynchronizedV1(id); err != nil || found {
		t.Fatalf("abandoned token survived actual reload: %v %v", found, err)
	}
	if owned, err := current.NFTsByOwnerSynchronizedV1(record.RecipientPubKeyHash); err != nil || len(owned) != 0 {
		t.Fatalf("abandoned owner survived actual reload: %v %v", owned, err)
	}
	if err := os.WriteFile(runtime.Config.ChainFile, []byte("corrupt ownership reload file"), 0o600); err != nil {
		t.Fatal(err)
	}
	if changed, err := runtime.ReloadChainFromDiskIfChanged(); err == nil || changed {
		t.Fatal("invalid reload replaced runtime ownership")
	}
	afterChain, _ := runtime.Snapshot()
	after, err := afterChain.NFTOwnershipSnapshotSynchronizedV1()
	if err != nil || afterChain != current || !reflect.DeepEqual(before, after) {
		t.Fatalf("failed reload changed current ownership: %+v %v", after, err)
	}
}
