package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"strings"
	"testing"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	qconsensus "quantumcoin/consensus"
)

func TestNFTAwardApprovalsBindCanonicalRecordV1(t *testing.T) {
	record := NFTMintRecordV1{NetworkID: "test-only-record-approval", CampaignID: "telegram-monthly", AwardPeriod: "2026-10", Serial: 1, RecipientPubKeyHash: bytes.Repeat([]byte{0x11}, 20), MetadataHash: [32]byte{0x44}}
	commitment, err := NFTMintRecordDigestV1(record)
	if err != nil {
		t.Fatal(err)
	}
	decision := qconsensus.NFTAwardDecisionV1{NetworkID: record.NetworkID, LegacyGenesisHex: strings.Repeat("ab", 32), RecordDigest: commitment}
	key, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := qconsensus.UncompressedPublicKeyV1(key)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := qconsensus.NFTAwardDecisionDigestV1(decision)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := qconsensus.SignTransactionDigestV1(key, digest)
	if err != nil {
		t.Fatal(err)
	}
	// Ephemeral test-only authority; this does not configure a real admin key.
	trust := qconsensus.NFTAwardTrustV1{NetworkID: record.NetworkID, LegacyGenesisHex: decision.LegacyGenesisHex, AdminPublicKeys: [][]byte{pub}, AdminQuorum: 1}
	approvals := []qconsensus.NFTAwardApprovalV1{{PublicKey: pub, Signature: sig}}
	if err := ValidateNFTAwardRecordApprovalV1(record, decision, trust, approvals); err != nil {
		t.Fatalf("valid approved record rejected: %v", err)
	}
	cases := []struct {
		name   string
		mutate func(*NFTMintRecordV1)
	}{
		{"recipient", func(r *NFTMintRecordV1) { r.RecipientPubKeyHash = bytes.Repeat([]byte{0x22}, 20) }},
		{"metadata", func(r *NFTMintRecordV1) { r.MetadataHash[0] ^= 1 }},
		{"period", func(r *NFTMintRecordV1) { r.AwardPeriod = "2026-11" }},
		{"campaign", func(r *NFTMintRecordV1) { r.CampaignID = "other-campaign" }},
		{"serial", func(r *NFTMintRecordV1) { r.Serial++ }},
		{"network", func(r *NFTMintRecordV1) { r.NetworkID = "other-network" }},
		{"invalid_recipient", func(r *NFTMintRecordV1) { r.RecipientPubKeyHash = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mutated := record
			tc.mutate(&mutated)
			if err := ValidateNFTAwardRecordApprovalV1(mutated, decision, trust, approvals); err == nil {
				t.Fatal("signature accepted for a modified NFT record")
			}
		})
	}
}
