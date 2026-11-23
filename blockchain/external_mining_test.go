package blockchain

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"quantumcoin/config"
	"quantumcoin/wallet"
	"strconv"
	"testing"
)

func solveExternalTemplate(t *testing.T, bc *Blockchain, block *Block) {
	t.Helper()
	for n := uint64(0); n < 1000000; n++ {
		header, err := bc.SerializeMiningHeader(block, n)
		if err != nil {
			t.Fatal(err)
		}
		h := sha256.Sum256(header)
		block.Hash = h[:]
		block.Nonce = int(n)
		if bc.ValidateBlockPoWAtHeightV1(block) == nil {
			return
		}
	}
	t.Fatal("no solution")
}

func TestExternalMiningUsesFeesAndClearsConfirmedMempool(t *testing.T) {
	t.Setenv("QC_DIFFICULTY_BITS", "8")
	bc, _, _, spend := newFeeConsensusTestStateV1(t)
	genesisHash := sha256.Sum256(bc.Blocks[0].Hash)
	bc.Blocks[0].Hash = genesisHash[:]
	if err := bc.AddTransaction(spend); err != nil {
		t.Fatal(err)
	}
	block, _, err := bc.PrepareExternalMiningTemplate(wallet.NewWallet().GetAddress(), 8)
	if err != nil {
		t.Fatal(err)
	}
	_, total, err := blockCoinbaseTotalV1(block, false)
	if err != nil || total != 53 {
		t.Fatalf("coinbase=%d err=%v", total, err)
	}
	solveExternalTemplate(t, bc, block)
	if err := bc.CommitValidatedBlockAndPersist(block, filepath.Join(t.TempDir(), "chain.dat")); err != nil {
		t.Fatal(err)
	}
	if len(bc.PendingTransactionsSnapshot()) != 0 || bc.TotalMinted() != 60 {
		t.Fatal("fee or mempool drift")
	}
}

func TestExternalMiningNFTActivationUsesV4HeaderAndAtomReward(t *testing.T) {
	bc, _, _ := nftLiveAdmissionFixtureV1(t, 1)
	diff := testExpectedNextDifficultyV1(t, bc)
	block, zero, err := bc.PrepareExternalMiningTemplate(wallet.NewWallet().GetAddress(), diff)
	if err != nil {
		t.Fatal(err)
	}
	if string(zero[:17]) != "QC_POW_HEADER_V4\x00" {
		t.Fatalf("wrong V4 domain: %x", zero[:17])
	}
	_, paid, err := blockCoinbaseTotalV1(block, false)
	expected, _ := bc.ExpectedSubsidyUnitsForHeightV1(1)
	if err != nil || paid != expected {
		t.Fatal("atom reward mismatch", paid, expected, err)
	}
	solveExternalTemplate(t, bc, block)
	if err := bc.CommitValidatedBlockAndPersist(block, filepath.Join(t.TempDir(), "chain.dat")); err != nil {
		t.Fatal(err)
	}
	if err := bc.ValidateBlockchainSynchronized(); err != nil {
		t.Fatal(err)
	}
}

func TestExternalMiningBrowserVectors(t *testing.T) {
	t.Setenv("QC_DIFFICULTY_BITS", "8")
	legacy := NewBlockchain(config.Default().InitialReward, config.Default().TotalSupply)
	nft, _, _ := nftLiveAdmissionFixtureV1(t, 1)
	vectors := []map[string]any{}
	for _, bc := range []*Blockchain{legacy, nft} {
		block, header, err := bc.PrepareExternalMiningTemplate(wallet.NewWallet().GetAddress(), testExpectedNextDifficultyV1(t, bc))
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range []uint64{0, 9007199254740993, MaxConsensusNonceV1} {
			encoded, err := bc.SerializeMiningHeader(block, n)
			if err != nil {
				t.Fatal(err)
			}
			if binary.BigEndian.Uint64(encoded[len(encoded)-8:]) != n {
				t.Fatal("nonce encoding")
			}
			hash := sha256.Sum256(encoded)
			vectors = append(vectors, map[string]any{"header": hex.EncodeToString(header), "nonce_offset": len(header) - 8, "nonce_encoding": "uint64-big-endian", "max_nonce": "9223372036854775807", "protocol": "QC_WEB_BLOCK_POW_V1", "nonce": stringUint(n), "hash": hex.EncodeToString(hash[:])})
		}
	}
	if filename := os.Getenv("QC_WEB_MINING_VECTOR_FILE"); filename != "" {
		data, _ := json.Marshal(vectors)
		if err := os.WriteFile(filename, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func stringUint(n uint64) string { return strconv.FormatUint(n, 10) }
