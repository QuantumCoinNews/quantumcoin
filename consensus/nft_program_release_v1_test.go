package consensus

import (
	"crypto/ecdsa"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func nftStartupProgramFixtureV1(t *testing.T) (NFTMonthlyProgramReleaseV1, NFTAwardTrustV1, []*ecdsa.PrivateKey) {
	t.Helper()
	d, trust, _, keys := nftAwardApprovalFixtureV1(t)
	release := NFTMonthlyProgramReleaseV1{Decision: NFTMonthlyProgramDecisionV1{NetworkID: d.NetworkID, LegacyGenesisHex: d.LegacyGenesisHex, CampaignID: "telegram-monthly", GameActivatedAt: time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC).Unix()}}
	nftStartupSignProgramV1(t, &release, keys[:2])
	return release, trust, keys
}

func nftStartupSignProgramV1(t *testing.T, r *NFTMonthlyProgramReleaseV1, keys []*ecdsa.PrivateKey) {
	t.Helper()
	digest, err := NFTMonthlyProgramDecisionDigestV1(r.Decision)
	if err != nil {
		t.Fatal(err)
	}
	r.Approvals = nil
	for _, key := range keys {
		pub, err := UncompressedPublicKeyV1(key)
		if err != nil {
			t.Fatal(err)
		}
		sig, err := SignTransactionDigestV1(key, digest)
		if err != nil {
			t.Fatal(err)
		}
		r.Approvals = append(r.Approvals, NFTAwardApprovalV1{PublicKey: pub, Signature: sig})
	}
}

func TestNFTStartupProgramReleaseBindsPeriodsAndDetachedTrustV1(t *testing.T) {
	r, trust, _ := nftStartupProgramFixtureV1(t)
	p, err := VerifyNFTMonthlyProgramReleaseV1(r, trust)
	if err != nil || !p.VerifiedV1() {
		t.Fatalf("verified program: %v", err)
	}
	policy := p.PolicySnapshotV1()
	if policy.FirstAwardPeriod != "2027-01" || p.GameActivatedAtV1() != r.Decision.GameActivatedAt {
		t.Fatal("game activation/first month changed")
	}
	for i := 1; i <= 12; i++ {
		month := time.Date(2027, time.Month(i), 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
		if serial, err := policy.SerialForAwardPeriodV1(month); err != nil || serial != uint32(i) {
			t.Fatalf("month %s: %d %v", month, serial, err)
		}
	}
	if _, err := policy.SerialForAwardPeriodV1("2028-01"); err == nil {
		t.Fatal("thirteenth month accepted")
	}
	key := append([]byte(nil), trust.AdminPublicKeys[0]...)
	trust.AdminPublicKeys[0][0] ^= 1
	r.Decision.GameActivatedAt++
	view := p.TrustSnapshotV1()
	view.AdminPublicKeys[0][1] ^= 1
	if string(p.TrustSnapshotV1().AdminPublicKeys[0]) != string(key) || p.PolicySnapshotV1() != policy {
		t.Fatal("caller mutation changed sealed program")
	}
	var decoded VerifiedNFTMonthlyProgramV1
	if err := json.Unmarshal([]byte(`{"verified":true,"GameActivatedAt":1}`), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.VerifiedV1() {
		t.Fatal("JSON manufactured verified program")
	}
}

func TestNFTStartupProgramReleaseRejectsTamperingQuorumAndReplayV1(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*testing.T, *NFTMonthlyProgramReleaseV1, *NFTAwardTrustV1, []*ecdsa.PrivateKey)
	}{
		{"changed_date", func(_ *testing.T, r *NFTMonthlyProgramReleaseV1, _ *NFTAwardTrustV1, _ []*ecdsa.PrivateKey) {
			r.Decision.GameActivatedAt++
		}},
		{"changed_campaign", func(_ *testing.T, r *NFTMonthlyProgramReleaseV1, _ *NFTAwardTrustV1, _ []*ecdsa.PrivateKey) {
			r.Decision.CampaignID = "other"
		}},
		{"changed_network", func(_ *testing.T, r *NFTMonthlyProgramReleaseV1, _ *NFTAwardTrustV1, _ []*ecdsa.PrivateKey) {
			r.Decision.NetworkID = "other"
		}},
		{"changed_genesis", func(_ *testing.T, r *NFTMonthlyProgramReleaseV1, _ *NFTAwardTrustV1, _ []*ecdsa.PrivateKey) {
			r.Decision.LegacyGenesisHex = strings.Repeat("cd", 32)
		}},
		{"missing_date", func(_ *testing.T, r *NFTMonthlyProgramReleaseV1, _ *NFTAwardTrustV1, _ []*ecdsa.PrivateKey) {
			r.Decision.GameActivatedAt = 0
		}},
		{"negative_date", func(_ *testing.T, r *NFTMonthlyProgramReleaseV1, _ *NFTAwardTrustV1, _ []*ecdsa.PrivateKey) {
			r.Decision.GameActivatedAt = -1
		}},
		{"insufficient_quorum", func(_ *testing.T, r *NFTMonthlyProgramReleaseV1, _ *NFTAwardTrustV1, _ []*ecdsa.PrivateKey) {
			r.Approvals = r.Approvals[:1]
		}},
		{"duplicate_signer", func(_ *testing.T, r *NFTMonthlyProgramReleaseV1, _ *NFTAwardTrustV1, _ []*ecdsa.PrivateKey) {
			r.Approvals[1] = r.Approvals[0]
		}},
		{"unknown_valid_signer", func(t *testing.T, r *NFTMonthlyProgramReleaseV1, _ *NFTAwardTrustV1, keys []*ecdsa.PrivateKey) {
			nftStartupSignProgramV1(t, r, keys[1:])
		}},
		{"zero_quorum", func(_ *testing.T, _ *NFTMonthlyProgramReleaseV1, trust *NFTAwardTrustV1, _ []*ecdsa.PrivateKey) {
			trust.AdminQuorum = 0
		}},
		{"award_signature_domain", func(t *testing.T, r *NFTMonthlyProgramReleaseV1, trust *NFTAwardTrustV1, keys []*ecdsa.PrivateKey) {
			d := NFTAwardDecisionV1{NetworkID: trust.NetworkID, LegacyGenesisHex: trust.LegacyGenesisHex, RecordDigest: [32]byte{1}}
			digest, err := NFTAwardDecisionDigestV1(d)
			if err != nil {
				t.Fatal(err)
			}
			for i, key := range keys[:2] {
				sig, err := SignTransactionDigestV1(key, digest)
				if err != nil {
					t.Fatal(err)
				}
				r.Approvals[i].Signature = sig
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, trust, keys := nftStartupProgramFixtureV1(t)
			tc.mutate(t, &r, &trust, keys)
			p, err := VerifyNFTMonthlyProgramReleaseV1(r, trust)
			if err == nil || p.VerifiedV1() {
				t.Fatal("invalid program verified")
			}
		})
	}
}

func TestNFTStartupProgramManifestRejectsSelfDeclaredTrustAndMalformedFilesV1(t *testing.T) {
	r, trust, _ := nftStartupProgramFixtureV1(t)
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), NFTMonthlyProgramManifestNameV1)
	write := func(b []byte) {
		t.Helper()
		if err := os.WriteFile(path, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(data)
	if p, err := LoadNFTMonthlyProgramManifestV1(path, trust); err != nil || !p.VerifiedV1() {
		t.Fatalf("valid manifest: %v", err)
	}
	for name, b := range map[string][]byte{
		"trust_override":  append(append([]byte(nil), data[:len(data)-1]...), []byte(`,"Trust":{"AdminQuorum":0}}`)...),
		"trailing_object": append(append([]byte(nil), data...), []byte(`{}`)...),
		"truncated":       data[:len(data)/2],
		"oversized":       []byte(strings.Repeat(" ", 65537)),
	} {
		t.Run(name, func(t *testing.T) {
			write(b)
			if p, err := LoadNFTMonthlyProgramManifestV1(path, trust); err == nil || p.VerifiedV1() {
				t.Fatal("malformed manifest accepted")
			}
		})
	}
	if CanonicalNFTAwardTrustV1() != nil {
		t.Fatal("test installed production NFT authorities")
	}
}
