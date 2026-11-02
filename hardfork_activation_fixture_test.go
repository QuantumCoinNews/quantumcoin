package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"quantumcoin/consensus"
)

// These ephemeral authorities and artifacts exist only in tests.
func signActivationReleaseFixtureV1(t *testing.T, d consensus.HardforkDecisionV1) consensus.HardforkReleaseV1 {
	t.Helper()
	payload, err := consensus.CanonicalHardforkDecisionPayloadV1(d)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(payload)
	approvals := make([]consensus.HardforkApprovalV1, 2)
	for i := range approvals {
		key, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		pub, err := consensus.UncompressedPublicKeyV1(key)
		if err != nil {
			t.Fatal(err)
		}
		sig, err := consensus.SignTransactionDigestV1(key, digest)
		if err != nil {
			t.Fatal(err)
		}
		approvals[i] = consensus.HardforkApprovalV1{PublicKey: pub, Signature: sig}
	}
	return consensus.HardforkReleaseV1{
		Trust: consensus.HardforkTrustV1{NetworkID: d.NetworkID, LegacyGenesisHex: d.LegacyGenesisHex,
			Authorities: consensus.HardforkAuthoritySetV1{
				FounderPublicKeys: [][]byte{approvals[0].PublicKey}, AdminPublicKeys: [][]byte{approvals[1].PublicKey},
				FounderQuorum: 1, AdminQuorum: 1,
			}},
		Decision:  d,
		Approvals: consensus.HardforkApprovalSetV1{Founder: approvals[:1], Admin: approvals[1:]},
	}
}

func verifiedActivationFixtureV1(t *testing.T, p consensus.ActivationPolicyV1) consensus.ActivationPolicyV1 {
	t.Helper()
	if !p.Enabled {
		return p
	}
	snapshot, binary := []byte("test-only audited snapshot"), []byte("test-only node artifact")
	sh, bh := sha256.Sum256(snapshot), sha256.Sum256(binary)
	d := consensus.HardforkDecisionV1{
		NetworkID: p.Plan.NetworkID, LegacyGenesisHex: p.Plan.LegacyGenesisHex,
		ActivationHeight: p.Plan.ActivationHeight, TargetProtocol: p.Plan.EffectiveTargetProtocol(),
		Reasons:        []string{"NETWORK_WIDE_PROTOCOL_UPGRADE_WITH_PUBLIC_NOTICE"},
		SnapshotSHA256: hex.EncodeToString(sh[:]), ReleaseBinarySHA256: hex.EncodeToString(bh[:]),
	}
	result, err := consensus.VerifyHardforkActivationV1(signActivationReleaseFixtureV1(t, d), snapshot, binary)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
