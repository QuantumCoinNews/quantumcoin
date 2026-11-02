package blockchain

import (
	"bytes"
	"encoding/hex"
	"path/filepath"
	"reflect"
	"testing"

	qconsensus "quantumcoin/consensus"
)

func nftAwardProofBlockFixtureV1(t *testing.T) (*Blockchain, *Block) {
	t.Helper()
	chain, record, decision, trust, approvals := nftLocalTrustFixtureV1(t)
	if err := chain.BindLocalNFTAwardTrustSynchronizedV1(record.NetworkID, trust); err != nil {
		t.Fatal(err)
	}
	block, err := NewNFTBlockWithAwardProofV1(1, nil, chain.Blocks[0].Hash, "test-only-miner", 1,
		[]NFTMintRecordV1{record}, []NFTAwardProofV1{{Decision: decision, Approvals: approvals}})
	if err != nil {
		t.Fatal(err)
	}
	return chain, block
}

func TestNFTAwardProofBlockBindsEvidenceBeforePoWV1(t *testing.T) {
	chain, block := nftAwardProofBlockFixtureV1(t)
	if err := ValidateBlockPoWV4(block); err != nil {
		t.Fatalf("clean V4 proof block: %v", err)
	}
	proof := block.NFTAwardProofsV1[0]
	if err := chain.ValidateNFTAwardWithLocalTrustSynchronizedV1(block.NFTMintRecordsV1[0], proof.Decision, proof.Approvals); err != nil {
		t.Fatal(err)
	}
	header, err := canonicalPoWHeaderV4(block)
	if err != nil {
		t.Fatal(err)
	}
	want, err := NFTBlockCommitmentDigestV1(block)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(header.NFTRecordDigest, want[:]) {
		t.Fatal("PoW adapter omitted award proof commitment")
	}
	mutations := []struct {
		name   string
		change func(*Block)
	}{
		{"genesis", func(b *Block) {
			b.NFTAwardProofsV1[0].Decision.LegacyGenesisHex = hex.EncodeToString(bytes.Repeat([]byte{0x77}, 32))
		}},
		{"signature", func(b *Block) {
			b.NFTAwardProofsV1[0].Approvals[0].Signature[len(b.NFTAwardProofsV1[0].Approvals[0].Signature)-1] ^= 1
		}},
		{"signer_order", func(b *Block) { a := b.NFTAwardProofsV1[0].Approvals; a[0], a[1] = a[1], a[0] }},
		{"removed_approval", func(b *Block) { b.NFTAwardProofsV1[0].Approvals = b.NFTAwardProofsV1[0].Approvals[:1] }},
		{"removed_proof", func(b *Block) { b.NFTAwardProofsV1 = nil }},
		{"recipient", func(b *Block) { b.NFTMintRecordsV1[0].RecipientPubKeyHash[0] ^= 1 }},
		{"metadata", func(b *Block) { b.NFTMintRecordsV1[0].MetadataHash[0] ^= 1 }},
	}
	for _, tc := range mutations {
		t.Run(tc.name, func(t *testing.T) {
			changed := cloneBlockV1(block)
			tc.change(changed)
			if err := ValidateBlockPoWV4(changed); err == nil {
				t.Fatal("stored PoW accepted changed NFT evidence")
			}
		})
	}
}

func TestNFTAwardProofBlockCopiesCallerEvidenceV1(t *testing.T) {
	chain, record, decision, trust, approvals := nftLocalTrustFixtureV1(t)
	if err := chain.BindLocalNFTAwardTrustSynchronizedV1(record.NetworkID, trust); err != nil {
		t.Fatal(err)
	}
	records := []NFTMintRecordV1{record}
	proofs := []NFTAwardProofV1{{Decision: decision, Approvals: approvals}}
	previous := append([]byte(nil), chain.Blocks[0].Hash...)
	block, err := NewNFTBlockWithAwardProofV1(1, nil, previous, "test-only-miner", 1, records, proofs)
	if err != nil {
		t.Fatal(err)
	}
	records[0].RecipientPubKeyHash[0] ^= 1
	proofs[0].Decision.NetworkID = "caller-mutation"
	approvals[0].PublicKey[0] ^= 1
	approvals[0].Signature[0] ^= 1
	previous[0] ^= 1
	if err := ValidateBlockPoWV4(block); err != nil {
		t.Fatalf("caller mutation changed mined block: %v", err)
	}
	p := block.NFTAwardProofsV1[0]
	if err := chain.ValidateNFTAwardWithLocalTrustSynchronizedV1(block.NFTMintRecordsV1[0], p.Decision, p.Approvals); err != nil {
		t.Fatal(err)
	}
}

func TestNFTAwardProofBlockDeepCopyAndSnapshotV1(t *testing.T) {
	chain, block := nftAwardProofBlockFixtureV1(t)
	copy := cloneBlockV1(block)
	copy.NFTAwardProofsV1[0].Approvals[0].PublicKey[0] ^= 1
	copy.NFTAwardProofsV1[0].Approvals[0].Signature[0] ^= 1
	copy.NFTAwardProofsV1[0].Decision.NetworkID = "clone-mutation"
	if err := ValidateBlockPoWV4(block); err != nil {
		t.Fatalf("clone mutated source: %v", err)
	}
	copy = cloneBlockV1(block)
	block.NFTAwardProofsV1[0].Approvals[0].Signature[0] ^= 1
	if err := ValidateBlockPoWV4(copy); err != nil {
		t.Fatalf("source mutated clone: %v", err)
	}
	chain.Blocks = append(chain.Blocks, copy) // Test-only, not live admission.
	snapshot := chain.AllBlocksSnapshot()
	snapshot[1].NFTAwardProofsV1[0].Approvals[0].Signature[0] ^= 1
	if err := ValidateBlockPoWV4(chain.Blocks[1]); err != nil {
		t.Fatalf("snapshot mutated live block: %v", err)
	}
}

func TestNFTAwardProofBlockSurvivesGobWithoutImportingTrustV1(t *testing.T) {
	chain, block := nftAwardProofBlockFixtureV1(t)
	decoded := DeserializeBlock(block.Serialize())
	if !reflect.DeepEqual(block.NFTAwardProofsV1, decoded.NFTAwardProofsV1) {
		t.Fatal("block gob dropped evidence")
	}
	if err := ValidateBlockPoWV4(decoded); err != nil {
		t.Fatal(err)
	}
	chain.Blocks = append(chain.Blocks, block) // Test-only serialization fixture.
	loaded := DeserializeBlockchain(SerializeBlockchain(chain))
	if loaded.NFTAwardTrustBoundSnapshotV1() {
		t.Fatal("peer chain imported local authorities")
	}
	if !reflect.DeepEqual(block.NFTAwardProofsV1, loaded.Blocks[1].NFTAwardProofsV1) {
		t.Fatal("chain gob dropped evidence")
	}
	if err := ValidateBlockPoWV4(loaded.Blocks[1]); err != nil {
		t.Fatal(err)
	}
}

func TestNFTAwardProofBlockRejectsMismatchedEvidenceV1(t *testing.T) {
	_, block := nftAwardProofBlockFixtureV1(t)
	cases := []struct {
		name   string
		change func(*Block)
	}{
		{"orphan", func(b *Block) { b.NFTMintRecordsV1 = nil }},
		{"extra_proof", func(b *Block) { b.NFTAwardProofsV1 = append(b.NFTAwardProofsV1, b.NFTAwardProofsV1[0]) }},
		{"no_approvals", func(b *Block) { b.NFTAwardProofsV1[0].Approvals = nil }},
		{"different_network", func(b *Block) { b.NFTAwardProofsV1[0].Decision.NetworkID = "different-network" }},
		{"wrong_record", func(b *Block) { b.NFTAwardProofsV1[0].Decision.RecordDigest[0] ^= 1 }},
		{"invalid_key", func(b *Block) { b.NFTAwardProofsV1[0].Approvals[0].PublicKey = []byte{1} }},
		{"invalid_signature", func(b *Block) { b.NFTAwardProofsV1[0].Approvals[0].Signature = []byte{1} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			invalid := cloneBlockV1(block)
			tc.change(invalid)
			if _, err := NFTBlockCommitmentDigestV1(invalid); err == nil {
				t.Fatal("invalid evidence representation accepted")
			}
		})
	}
	if _, err := NFTBlockCommitmentDigestV1(nil); err == nil {
		t.Fatal("nil block accepted")
	}
}

func TestNFTAwardProofBlockPreservesRecordOnlyPoWV1(t *testing.T) {
	_, block := nftAwardProofBlockFixtureV1(t)
	for _, records := range [][]NFTMintRecordV1{nil, block.NFTMintRecordsV1} {
		b := &Block{NFTMintRecordsV1: records}
		oldDigest, err := NFTMintRecordsDigestV1(records)
		if err != nil {
			t.Fatal(err)
		}
		newDigest, err := NFTBlockCommitmentDigestV1(b)
		if err != nil {
			t.Fatal(err)
		}
		if oldDigest != newDigest {
			t.Fatal("existing record-only digest changed")
		}
	}
}

func TestNFTAwardProofBlockOrphanRejectedByAdmissionAndLoadV1(t *testing.T) {
	_, block := nftAwardProofBlockFixtureV1(t)
	chain := NewBlockchain(1, 0)
	if !chain.IsValidChainSynchronized() {
		t.Fatal("clean positive-control chain invalid")
	}
	cleanFile := filepath.Join(t.TempDir(), "clean.gob")
	if err := chain.SaveToFile(cleanFile); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBlockchainFromFile(cleanFile); err != nil {
		t.Fatalf("clean file rejected: %v", err)
	}
	chain.Blocks[0].NFTAwardProofsV1 = cloneNFTAwardProofsV1(block.NFTAwardProofsV1)
	for _, protocol := range []qconsensus.ProtocolVersion{qconsensus.ProtocolLegacyV1, qconsensus.ProtocolNFTV4} {
		if err := validateNFTMintAuthorizationV1(chain.Blocks[0], protocol); err == nil {
			t.Fatal("orphan proof bypassed NFT authorization")
		}
	}
	if chain.IsValidChainSynchronized() {
		t.Fatal("orphan proof accepted in chain")
	}
	if _, err := chain.ValidateCanonicalBlockEnvelopeV1(chain.Blocks[0], 0); err == nil {
		t.Fatal("orphan proof accepted by canonical envelope")
	}
	badFile := filepath.Join(t.TempDir(), "orphan.gob")
	if err := chain.SaveToFile(badFile); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBlockchainFromFile(badFile); err == nil {
		t.Fatal("orphan proof bypassed load guard")
	}
}

func TestNFTAwardProofBlockDoesNotEnableLiveAdmissionV1(t *testing.T) {
	chain, block := nftAwardProofBlockFixtureV1(t)
	if err := validateNFTMintAuthorizationV1(block, qconsensus.ProtocolNFTV4); err == nil {
		t.Fatal("proof alone enabled live admission")
	}
	before := chain.AllBlocksSnapshot()
	if err := chain.AddBlockFromPeerSynchronizedV2(block); err == nil {
		t.Fatal("peer path admitted NFT proof before runtime integration")
	}
	if !reflect.DeepEqual(before, chain.AllBlocksSnapshot()) {
		t.Fatal("rejected proof changed chain")
	}
}

func TestNFTAwardProofBlockConstructorRejectsMissingInputsV1(t *testing.T) {
	_, block := nftAwardProofBlockFixtureV1(t)
	if _, err := NewNFTBlockWithAwardProofV1(0, nil, block.PrevHash, "test", 1, block.NFTMintRecordsV1, block.NFTAwardProofsV1); err == nil {
		t.Fatal("genesis award created")
	}
	if _, err := NewNFTBlockWithAwardProofV1(1, nil, block.PrevHash, "test", 1, block.NFTMintRecordsV1, nil); err == nil {
		t.Fatal("missing proof accepted")
	}
	if _, err := NewNFTBlockWithAwardProofV1(1, nil, block.PrevHash, "test", 1, nil, block.NFTAwardProofsV1); err == nil {
		t.Fatal("missing records accepted")
	}
}
