package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func nftTrustedLoadFixtureV1(t *testing.T) (*Blockchain, *Blockchain, NFTMintRecordV1, []*ecdsa.PrivateKey) {
	t.Helper()
	trusted, p, keys := nftLiveAdmissionFixtureV1(t, 1)
	trusted.SetCoinbaseMaturity(2)
	producer := DeserializeBlockchain(SerializeBlockchain(trusted))
	nftLiveAdmissionRebindV1(t, producer, trusted)
	producer.SetCoinbaseMaturity(2)
	record := nftMonthlyPolicyRecordV1(p, 0)
	block := nftLiveAdmissionBlockV1(t, trusted, record, keys, 0)
	if err := producer.AddBlockFromPeerSynchronizedV2(block); err != nil {
		t.Fatal(err)
	}
	return trusted, producer, record, keys
}

func nftTrustedLoadWriteV1(t *testing.T, chain *Blockchain) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "trusted-load.dat")
	if err := chain.SaveToFile(path); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestNFTTrustedLoadRestoresApprovedChainAndRebuildsStateV1(t *testing.T) {
	trusted, producer, record, _ := nftTrustedLoadFixtureV1(t)
	wire := DeserializeBlockchain(SerializeBlockchain(producer))
	wire.UTXO = map[string][]TransactionOutput{"forged-free-balance": {{Amount: 123456, PubKeyHash: []byte("forged")}}}
	path := nftTrustedLoadWriteV1(t, wire)
	loaded, err := LoadBlockchainFromFileWithLocalContextV1(path, trusted)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.BestHeightSnapshot() != 1 || loaded.ProtocolAtHeightV1(1) != producer.ProtocolAtHeightV1(1) || !loaded.IsValidChainSynchronized() {
		t.Fatal("trusted reload lost approved chain/activation")
	}
	if !reflect.DeepEqual(loaded.UTXO, producer.UTXO) || loaded.coinbaseMaturity != 2 {
		t.Fatal("file UTXO was trusted or local maturity was lost")
	}
	if loaded.confirmedTxIndex[hex.EncodeToString(producer.Blocks[1].Transactions[0].ID)] != 1 {
		t.Fatal("confirmed transaction index was not rebuilt")
	}
	if len(loaded.pendingTxs) != 0 || !loaded.nftAwardTrust.bound || !loaded.nftMonthlyProgram.bound {
		t.Fatal("loaded local runtime context is incomplete")
	}
	proof := producer.AllBlocksSnapshot()[1].NFTAwardProofsV1[0]
	before := loaded.AllBlocksSnapshot()
	if _, err := loaded.MineBlockWithNFTAwardSynchronizedV1("test-only-reloaded-miner", testExpectedNextDifficultyV1(t, loaded), record, proof); err == nil || !strings.Contains(err.Error(), "duplicate NFT award period") {
		t.Fatalf("restart did not reject the same month through canonical history: %v", err)
	}
	if !reflect.DeepEqual(before, loaded.AllBlocksSnapshot()) {
		t.Fatal("rejected repeat award changed restored state")
	}
	original := trusted.nftAwardTrust.trust.AdminPublicKeys[0][0]
	trusted.nftAwardTrust.trust.AdminPublicKeys[0][0] ^= 1
	if loaded.nftAwardTrust.trust.AdminPublicKeys[0][0] != original {
		t.Fatal("restored authority keys alias source runtime")
	}
	trusted.nftAwardTrust.trust.AdminPublicKeys[0][0] = original
	if trusted.BestHeightSnapshot() != 0 {
		t.Fatal("loading changed the source runtime chain")
	}
}

func TestNFTTrustedLoadRejectsMissingContextAndUnauthorizedEvidenceV1(t *testing.T) {
	for _, name := range []string{"nil_context", "unbound_context", "activation_and_trust_without_program", "unknown_signers", "missing_proof", "genesis_award"} {
		t.Run(name, func(t *testing.T) {
			trusted, producer, record, _ := nftTrustedLoadFixtureV1(t)
			target := trusted
			wire := DeserializeBlockchain(SerializeBlockchain(producer))
			if name == "nil_context" {
				target = nil
			}
			if name == "unbound_context" || name == "activation_and_trust_without_program" {
				target = DeserializeBlockchain(SerializeBlockchain(trusted))
				if name == "activation_and_trust_without_program" {
					if err := target.BindActivationPolicySynchronizedV1(trusted.activationPolicy.policy); err != nil {
						t.Fatal(err)
					}
					if err := target.BindLocalNFTAwardTrustSynchronizedV1(record.NetworkID, trusted.nftAwardTrust.trust); err != nil {
						t.Fatal(err)
					}
				}
			}
			if name == "unknown_signers" {
				_, _, rogue := nftLiveAdmissionFixtureV1(t, 1)
				wire.Blocks[1] = nftLiveAdmissionBlockV1(t, trusted, record, rogue, 0)
			}
			if name == "missing_proof" {
				wire.Blocks[1].NFTAwardProofsV1 = nil
			}
			if name == "genesis_award" {
				wire.Blocks[0].NFTMintRecordsV1 = []NFTMintRecordV1{record}
			}
			loaded, err := LoadBlockchainFromFileWithLocalContextV1(nftTrustedLoadWriteV1(t, wire), target)
			if err == nil || loaded != nil {
				t.Fatal("file supplied missing authority/program or unauthorized awards")
			}
			if trusted.BestHeightSnapshot() != 0 {
				t.Fatal("rejected file changed the trusted runtime")
			}
		})
	}
}

func TestNFTTrustedLoadRejectsCorruptAndNoncanonicalFilesV1(t *testing.T) {
	for _, name := range []string{"malformed_gob", "truncated_gob", "empty_chain", "supply_drift", "genesis_metadata_drift", "inflated_coinbase_with_valid_pow"} {
		t.Run(name, func(t *testing.T) {
			trusted, producer, record, keys := nftTrustedLoadFixtureV1(t)
			wire := DeserializeBlockchain(SerializeBlockchain(producer))
			expected := ""
			switch name {
			case "empty_chain":
				wire.Blocks = nil
			case "supply_drift":
				wire.TotalSupply++
				expected = "total supply differs"
			case "genesis_metadata_drift":
				wire.Blocks[0].Metadata = map[string]string{"untrusted": "changed"}
				expected = "genesis metadata mismatch"
			case "inflated_coinbase_with_valid_pow":
				wire.Blocks[1] = nftLiveAdmissionBlockV1(t, trusted, record, keys, 1)
				if err := ValidateBlockPoWV4(wire.Blocks[1]); err != nil {
					t.Fatal(err)
				}
				expected = "coinbase mismatch"
			}
			var data []byte
			if name == "malformed_gob" {
				data = []byte("not a gob chain")
			} else if name == "truncated_gob" {
				data = SerializeBlockchain(wire)
				data = data[:len(data)/2]
			} else {
				data = SerializeBlockchain(wire)
			}
			path := filepath.Join(t.TempDir(), "corrupt.dat")
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			loaded, err := LoadBlockchainFromFileWithLocalContextV1(path, trusted)
			if err == nil || loaded != nil {
				t.Fatal("corrupt/noncanonical persisted chain was accepted")
			}
			if expected != "" && !strings.Contains(err.Error(), expected) {
				t.Fatalf("rejection did not reach expected file/consensus check: %v", err)
			}
		})
	}
}

func TestNFTTrustedLoadAllowsGenesisAndEqualHeightWithoutWeakeningReplacementV1(t *testing.T) {
	legacy := NewBlockchain(1, 0)
	path := nftTrustedLoadWriteV1(t, legacy)
	restored, err := LoadBlockchainFromFileWithLocalContextV1(path, legacy)
	if err != nil || restored.BestHeightSnapshot() != 0 {
		t.Fatalf("trusted genesis-only load: %v", err)
	}
	_, producer, _, _ := nftTrustedLoadFixtureV1(t)
	path = nftTrustedLoadWriteV1(t, producer)
	equal, err := LoadBlockchainFromFileWithLocalContextV1(path, producer)
	if err != nil || !bytes.Equal(equal.AllBlocksSnapshot()[1].Hash, producer.AllBlocksSnapshot()[1].Hash) {
		t.Fatalf("equal-height trusted load: %v", err)
	}
	if err := producer.ValidateReplacementChainSynchronized(producer.AllBlocksSnapshot()); !errors.Is(err, ErrIncomingChainNotLonger) {
		t.Fatalf("load refactor weakened replacement length rule: %v", err)
	}
	if loaded, err := LoadBlockchainFromFile(path); err == nil || loaded != nil {
		t.Fatal("context-free loader acquired NFT authority from the file")
	}
}
