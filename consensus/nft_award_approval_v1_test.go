package consensus

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"strings"
	"testing"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

func nftAwardApprovalFixtureV1(t *testing.T) (NFTAwardDecisionV1, NFTAwardTrustV1, []NFTAwardApprovalV1, []*ecdsa.PrivateKey) {
	t.Helper()
	decision := NFTAwardDecisionV1{
		NetworkID: "test-only-nft-award", LegacyGenesisHex: strings.Repeat("ab", 32), RecordDigest: [32]byte{0x55},
	}
	digest, err := NFTAwardDecisionDigestV1(decision)
	if err != nil {
		t.Fatal(err)
	}
	trust := NFTAwardTrustV1{NetworkID: decision.NetworkID, LegacyGenesisHex: decision.LegacyGenesisHex, AdminQuorum: 2}
	var approvals []NFTAwardApprovalV1
	var keys []*ecdsa.PrivateKey
	for i := 0; i < 3; i++ {
		key, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		pub, err := UncompressedPublicKeyV1(key)
		if err != nil {
			t.Fatal(err)
		}
		sig, err := SignTransactionDigestV1(key, digest)
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, key)
		approvals = append(approvals, NFTAwardApprovalV1{PublicKey: pub, Signature: sig})
		if i < 2 {
			trust.AdminPublicKeys = append(trust.AdminPublicKeys, pub)
		}
	}
	return decision, trust, approvals, keys
}

func TestNFTAwardApprovalsAcceptValidIndependentQuorumV1(t *testing.T) {
	d, trust, all, _ := nftAwardApprovalFixtureV1(t)
	for _, approvals := range [][]NFTAwardApprovalV1{all[:2], {all[1], all[0]}} {
		if err := ValidateNFTAwardApprovalsV1(d, trust, approvals, d.RecordDigest); err != nil {
			t.Fatal(err)
		}
	}
	upper := d
	upper.LegacyGenesisHex = strings.ToUpper(d.LegacyGenesisHex)
	lowerDigest, err := NFTAwardDecisionDigestV1(d)
	if err != nil {
		t.Fatal(err)
	}
	upperDigest, err := NFTAwardDecisionDigestV1(upper)
	if err != nil {
		t.Fatal(err)
	}
	if lowerDigest != upperDigest {
		t.Fatal("genesis hex case changed canonical approval digest")
	}
	if err := ValidateNFTAwardApprovalsV1(upper, trust, all[:2], d.RecordDigest); err != nil {
		t.Fatal(err)
	}
}

func TestNFTAwardApprovalsRejectForgeryAndReplayV1(t *testing.T) {
	d, trust, all, _ := nftAwardApprovalFixtureV1(t)
	type attack func(*NFTAwardDecisionV1, *NFTAwardTrustV1, *[]NFTAwardApprovalV1, *[32]byte)
	cases := []struct {
		name   string
		mutate attack
	}{
		{"missing_proof", func(_ *NFTAwardDecisionV1, _ *NFTAwardTrustV1, a *[]NFTAwardApprovalV1, _ *[32]byte) { *a = nil }},
		{"insufficient_quorum", func(_ *NFTAwardDecisionV1, _ *NFTAwardTrustV1, a *[]NFTAwardApprovalV1, _ *[32]byte) { *a = (*a)[:1] }},
		{"duplicate_signer", func(_ *NFTAwardDecisionV1, _ *NFTAwardTrustV1, a *[]NFTAwardApprovalV1, _ *[32]byte) {
			(*a)[1] = (*a)[0]
		}},
		{"unauthorized_valid_signer", func(_ *NFTAwardDecisionV1, _ *NFTAwardTrustV1, a *[]NFTAwardApprovalV1, _ *[32]byte) {
			(*a)[1] = all[2]
		}},
		{"malformed_signature", func(_ *NFTAwardDecisionV1, _ *NFTAwardTrustV1, a *[]NFTAwardApprovalV1, _ *[32]byte) {
			(*a)[0].Signature = []byte{1}
		}},
		{"actual_record_changed", func(_ *NFTAwardDecisionV1, _ *NFTAwardTrustV1, _ *[]NFTAwardApprovalV1, expected *[32]byte) {
			expected[0] ^= 1
		}},
		{"signed_record_substituted", func(d *NFTAwardDecisionV1, _ *NFTAwardTrustV1, _ *[]NFTAwardApprovalV1, expected *[32]byte) {
			d.RecordDigest[0] ^= 1
			*expected = d.RecordDigest
		}},
		{"different_local_network", func(_ *NFTAwardDecisionV1, trust *NFTAwardTrustV1, _ *[]NFTAwardApprovalV1, _ *[32]byte) {
			trust.NetworkID = "other-network"
		}},
		{"different_local_genesis", func(_ *NFTAwardDecisionV1, trust *NFTAwardTrustV1, _ *[]NFTAwardApprovalV1, _ *[32]byte) {
			trust.LegacyGenesisHex = strings.Repeat("cd", 32)
		}},
		{"cross_network_signature_replay", func(d *NFTAwardDecisionV1, trust *NFTAwardTrustV1, _ *[]NFTAwardApprovalV1, _ *[32]byte) {
			d.NetworkID = "other-network"
			trust.NetworkID = d.NetworkID
		}},
		{"cross_genesis_signature_replay", func(d *NFTAwardDecisionV1, trust *NFTAwardTrustV1, _ *[]NFTAwardApprovalV1, _ *[32]byte) {
			d.LegacyGenesisHex = strings.Repeat("cd", 32)
			trust.LegacyGenesisHex = d.LegacyGenesisHex
		}},
		{"zero_quorum", func(_ *NFTAwardDecisionV1, trust *NFTAwardTrustV1, _ *[]NFTAwardApprovalV1, _ *[32]byte) {
			trust.AdminQuorum = 0
		}},
		{"impossible_quorum", func(_ *NFTAwardDecisionV1, trust *NFTAwardTrustV1, _ *[]NFTAwardApprovalV1, _ *[32]byte) {
			trust.AdminQuorum = 3
		}},
		{"duplicate_local_authority", func(_ *NFTAwardDecisionV1, trust *NFTAwardTrustV1, _ *[]NFTAwardApprovalV1, _ *[32]byte) {
			trust.AdminPublicKeys = [][]byte{trust.AdminPublicKeys[0], trust.AdminPublicKeys[0]}
		}},
		{"empty_local_trust", func(_ *NFTAwardDecisionV1, trust *NFTAwardTrustV1, _ *[]NFTAwardApprovalV1, _ *[32]byte) {
			trust.AdminPublicKeys = nil
		}},
		{"invalid_local_key", func(_ *NFTAwardDecisionV1, trust *NFTAwardTrustV1, _ *[]NFTAwardApprovalV1, _ *[32]byte) {
			trust.AdminPublicKeys = [][]byte{{1}, trust.AdminPublicKeys[1]}
		}},
		{"zero_record_commitment", func(d *NFTAwardDecisionV1, _ *NFTAwardTrustV1, _ *[]NFTAwardApprovalV1, expected *[32]byte) {
			d.RecordDigest = [32]byte{}
			*expected = d.RecordDigest
		}},
		{"invalid_genesis", func(d *NFTAwardDecisionV1, _ *NFTAwardTrustV1, _ *[]NFTAwardApprovalV1, _ *[32]byte) {
			d.LegacyGenesisHex = "not-a-genesis"
		}},
		{"invalid_network", func(d *NFTAwardDecisionV1, _ *NFTAwardTrustV1, _ *[]NFTAwardApprovalV1, _ *[32]byte) {
			d.NetworkID = "bad network"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decision, local, expected := d, trust, d.RecordDigest
			local.AdminPublicKeys = append([][]byte(nil), trust.AdminPublicKeys...)
			approvals := append([]NFTAwardApprovalV1(nil), all[:2]...)
			tc.mutate(&decision, &local, &approvals, &expected)
			if err := ValidateNFTAwardApprovalsV1(decision, local, approvals, expected); err == nil {
				t.Fatal("NFT award attack was accepted")
			}
		})
	}
}

func TestNFTAwardApprovalsRejectHardforkSignatureDomainV1(t *testing.T) {
	d, trust, _, keys := nftAwardApprovalFixtureV1(t)
	hardfork := HardforkDecisionV1{
		NetworkID: d.NetworkID, LegacyGenesisHex: d.LegacyGenesisHex, ActivationHeight: 1, TargetProtocol: ProtocolNFTV4,
		Reasons:        []string{"NETWORK_WIDE_PROTOCOL_UPGRADE_WITH_PUBLIC_NOTICE"},
		SnapshotSHA256: strings.Repeat("11", 32), ReleaseBinarySHA256: strings.Repeat("22", 32),
	}
	payload, err := CanonicalHardforkDecisionPayloadV1(hardfork)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(payload)
	var approvals []NFTAwardApprovalV1
	for _, key := range keys[:2] {
		pub, err := UncompressedPublicKeyV1(key)
		if err != nil {
			t.Fatal(err)
		}
		sig, err := SignTransactionDigestV1(key, digest)
		if err != nil {
			t.Fatal(err)
		}
		approvals = append(approvals, NFTAwardApprovalV1{PublicKey: pub, Signature: sig})
	}
	if err := ValidateNFTAwardApprovalsV1(d, trust, approvals, d.RecordDigest); err == nil {
		t.Fatal("hardfork signatures were accepted as NFT award approvals")
	}
}
