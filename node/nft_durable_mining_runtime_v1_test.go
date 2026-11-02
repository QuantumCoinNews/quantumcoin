package node

import (
	"bytes"
	"os"
	"reflect"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/wallet"
)

type nftDurableRuntimeEntryForTestV1 interface {
	MineNFTMonthlyAwardAndPersistV1(string, blockchain.NFTMintRecordV1, blockchain.NFTAwardProofV1) (*blockchain.Block, error)
}

func nftDurableRuntimeEntryV1(t *testing.T, r *Runtime) nftDurableRuntimeEntryForTestV1 {
	t.Helper()
	entry, ok := any(r).(nftDurableRuntimeEntryForTestV1)
	if !ok {
		t.Fatal("persistent NFT runtime entry is missing")
	}
	return entry
}

func nftDurableRuntimeFixtureV1(t *testing.T) (*Runtime, *nftStartupFixtureDataV1, blockchain.NFTAwardProofV1) {
	t.Helper()
	f := nftStartupFixtureV1(t)
	proof := f.producer.AllBlocksSnapshot()[1].NFTAwardProofsV1[0]
	// Retain this fixture's exact canonical genesis, without its already
	// minted award, so the new runtime itself must perform the durable mint.
	state := f.producer.StateSnapshot()
	genesis := &blockchain.Blockchain{Blocks: state.Blocks[:1], TotalSupply: state.TotalSupply}
	genesis.UpdateUTXOSet()
	if err := genesis.SaveToFile(f.cfg.ChainFile); err != nil {
		t.Fatal(err)
	}
	r, err := bootstrapWithNFTStartupV1(f.cfg, true, &f.release, f.snapshotDir, &f.trust, f.manifest)
	if err != nil {
		t.Fatal(err)
	}
	return r, f, proof
}

func nftDurableRuntimeReadV1(t *testing.T, filename string) []byte {
	t.Helper()
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestNFTDurableRuntimePersistsApprovedAwardThroughVerifiedStartupV1(t *testing.T) {
	r, f, proof := nftDurableRuntimeFixtureV1(t)
	program := r.NFTGameProgramSnapshotV1()
	block, err := nftDurableRuntimeEntryV1(t, r).MineNFTMonthlyAwardAndPersistV1(wallet.NewWallet().GetAddress(), f.record, proof)
	if err != nil {
		t.Fatal(err)
	}
	if r.Blockchain.BestHeightSnapshot() != 1 || r.NFTGameProgramSnapshotV1() != program {
		t.Fatal("runtime did not retain canonical award and verified program")
	}
	file := nftDurableRuntimeReadV1(t, f.cfg.ChainFile)
	// A second real bootstrap must verify the newly written award using the
	// same independently verified local program and signing authorities.
	restarted, err := bootstrapWithNFTStartupV1(f.cfg, true, &f.release, f.snapshotDir, &f.trust, f.manifest)
	if err != nil {
		t.Fatal(err)
	}
	owned, err := restarted.Blockchain.NFTOwnershipSnapshotSynchronizedV1()
	if err != nil || owned.Height != 1 || len(owned.NFTs) != 1 || !bytes.Equal(owned.TipHash, block.Hash) ||
		!reflect.DeepEqual(owned.NFTs[0].Record, f.record) {
		t.Fatalf("new runtime mint did not survive verified restart: %+v %v", owned, err)
	}
	if !bytes.Equal(file, nftDurableRuntimeReadV1(t, f.cfg.ChainFile)) {
		t.Fatal("verified restart wrote the minted chain file")
	}
}

func TestNFTDurableRuntimeRejectsReadOnlyMissingProgramAndInvalidEvidenceV1(t *testing.T) {
	for _, name := range []string{"read_only", "missing_program", "missing_config", "empty_filename", "changed_recipient"} {
		t.Run(name, func(t *testing.T) {
			r, f, proof := nftDurableRuntimeFixtureV1(t)
			chain := r.Blockchain
			state, file := chain.StateSnapshot(), nftDurableRuntimeReadV1(t, f.cfg.ChainFile)
			record := f.record
			filename := f.cfg.ChainFile
			switch name {
			case "read_only":
				r.SetAPIReadOnly(true)
			case "missing_program":
				r = NewRuntime(chain, f.cfg)
			case "missing_config":
				r.Config = nil
			case "empty_filename":
				copyConfig := *r.Config
				copyConfig.ChainFile = ""
				r.Config = &copyConfig
			case "changed_recipient":
				record.RecipientPubKeyHash = append([]byte(nil), record.RecipientPubKeyHash...)
				record.RecipientPubKeyHash[0] ^= 1
			}
			if _, err := nftDurableRuntimeEntryV1(t, r).MineNFTMonthlyAwardAndPersistV1(wallet.NewWallet().GetAddress(), record, proof); err == nil {
				t.Fatal("invalid runtime context/evidence minted an award")
			}
			if !reflect.DeepEqual(state, chain.StateSnapshot()) || !bytes.Equal(file, nftDurableRuntimeReadV1(t, filename)) {
				t.Fatal("rejected runtime award changed RAM or chain file")
			}
		})
	}
}
