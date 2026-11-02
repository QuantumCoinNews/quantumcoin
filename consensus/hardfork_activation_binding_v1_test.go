package consensus

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHardforkActivationRejectsUnsignedPlanV1(t *testing.T) {
	p := ActivationPolicyV1{Enabled: true, Plan: atomActivationTestPlanV1()}
	p.Plan.TargetProtocol = ProtocolAtomV3
	if err := p.Validate(); err == nil {
		t.Fatal("unsigned activation policy validated without decision, approvals or artifact verification")
	}
	if got := p.ProtocolAtHeight(p.Plan.ActivationHeight); got != ProtocolLegacyV1 {
		t.Fatalf("unsigned policy selected %s", got)
	}
}

func hardforkBindingReleaseFixtureV1(t *testing.T) (HardforkReleaseV1, []byte, []byte) {
	t.Helper()
	snapshot, binary := []byte("audited fixture state"), []byte("fixture node executable")
	d := hardforkArtifactDecisionV1(snapshot, binary)
	d.Reasons = append(d.Reasons, "CRITICAL_CONSENSUS_BUG")
	return signActivationReleaseFixtureV1(t, d), snapshot, binary
}

func TestHardforkActivationDerivesSignedPlanV1(t *testing.T) {
	r, s, b := hardforkBindingReleaseFixtureV1(t)
	p, err := VerifyHardforkActivationV1(r, s, b)
	if err != nil {
		t.Fatal(err)
	}
	if p.Plan.NetworkID != r.Decision.NetworkID || p.Plan.LegacyGenesisHex != r.Decision.LegacyGenesisHex ||
		p.Plan.ActivationHeight != r.Decision.ActivationHeight || p.Plan.TargetProtocol != r.Decision.TargetProtocol ||
		!p.Plan.PreserveHistory || !p.Plan.PreserveBalances {
		t.Fatalf("plan diverged from decision: %+v", p.Plan)
	}
	for _, h := range []int{r.Decision.ActivationHeight - 1, r.Decision.ActivationHeight, r.Decision.ActivationHeight + 1} {
		want := ProtocolLegacyV1
		if h >= r.Decision.ActivationHeight {
			want = r.Decision.TargetProtocol
		}
		if got := p.ProtocolAtHeight(h); got != want {
			t.Fatalf("height %d: got %s want %s", h, got, want)
		}
	}
	r.Decision.Reasons[0], r.Decision.Reasons[1] = r.Decision.Reasons[1], r.Decision.Reasons[0]
	p2, err := VerifyHardforkActivationV1(r, s, b)
	if err != nil || p2 != p {
		t.Fatalf("reason order changed plan: %v", err)
	}
	r.Trust.Authorities.FounderPublicKeys[0][0] ^= 1
	r.Decision.ActivationHeight++
	s[0] ^= 1
	b[0] ^= 1
	if err := p.Validate(); err != nil {
		t.Fatalf("caller mutation changed verified policy: %v", err)
	}
}

func TestHardforkActivationRejectsInvalidReleaseV1(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*HardforkReleaseV1, *[]byte, *[]byte)
	}{
		{"height", func(r *HardforkReleaseV1, s, b *[]byte) { r.Decision.ActivationHeight++ }},
		{"protocol", func(r *HardforkReleaseV1, s, b *[]byte) { r.Decision.TargetProtocol = ProtocolContinuityV2 }},
		{"network", func(r *HardforkReleaseV1, s, b *[]byte) { r.Decision.NetworkID = "other" }},
		{"genesis", func(r *HardforkReleaseV1, s, b *[]byte) { r.Decision.LegacyGenesisHex = strings.Repeat("dd", 32) }},
		{"snapshot", func(r *HardforkReleaseV1, s, b *[]byte) { (*s)[0] ^= 1 }},
		{"binary", func(r *HardforkReleaseV1, s, b *[]byte) { (*b)[0] ^= 1 }},
		{"missing_snapshot", func(r *HardforkReleaseV1, s, b *[]byte) { *s = nil }},
		{"missing_binary", func(r *HardforkReleaseV1, s, b *[]byte) { *b = nil }},
		{"founder_quorum", func(r *HardforkReleaseV1, s, b *[]byte) { r.Approvals.Founder = nil }},
		{"admin_quorum", func(r *HardforkReleaseV1, s, b *[]byte) { r.Approvals.Admin = nil }},
		{"unauthorized_signer", func(r *HardforkReleaseV1, s, b *[]byte) {
			r.Approvals.Admin[0].PublicKey = append([]byte(nil), r.Approvals.Founder[0].PublicKey...)
		}},
		{"duplicate_signer", func(r *HardforkReleaseV1, s, b *[]byte) {
			r.Approvals.Admin = append(r.Approvals.Admin, r.Approvals.Admin[0])
		}},
		{"duplicate_authority", func(r *HardforkReleaseV1, s, b *[]byte) {
			r.Trust.Authorities.AdminPublicKeys = append(r.Trust.Authorities.AdminPublicKeys, r.Trust.Authorities.AdminPublicKeys[0])
		}},
		{"wrong_signature", func(r *HardforkReleaseV1, s, b *[]byte) { r.Approvals.Admin[0].Signature[0] ^= 1 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, s, b := hardforkBindingReleaseFixtureV1(t)
			tc.mutate(&r, &s, &b)
			p, err := VerifyHardforkActivationV1(r, s, b)
			if err == nil || p != (ActivationPolicyV1{}) {
				t.Fatalf("invalid release returned usable policy: %+v, %v", p, err)
			}
		})
	}
}

func TestHardforkActivationRejectsPlanMutationV1(t *testing.T) {
	r, s, b := hardforkBindingReleaseFixtureV1(t)
	verified, err := VerifyHardforkActivationV1(r, s, b)
	if err != nil {
		t.Fatal(err)
	}
	mutations := []func(*ActivationPolicyV1){
		func(p *ActivationPolicyV1) { p.Plan.ActivationHeight++ },
		func(p *ActivationPolicyV1) { p.Plan.TargetProtocol = ProtocolContinuityV2 },
		func(p *ActivationPolicyV1) { p.Plan.NetworkID = "other" },
		func(p *ActivationPolicyV1) { p.Plan.LegacyGenesisHex = strings.Repeat("ee", 32) },
		func(p *ActivationPolicyV1) { p.Plan.PreserveHistory = false },
		func(p *ActivationPolicyV1) { p.Plan.PreserveBalances = false },
		func(p *ActivationPolicyV1) { p.Enabled = false },
	}
	for i, mutate := range mutations {
		p := verified
		mutate(&p)
		if p.Validate() == nil || p.ProtocolAtHeight(1000000) != ProtocolLegacyV1 {
			t.Fatalf("mutation %d retained activation authority", i)
		}
	}
	encoded, err := json.Marshal(verified)
	if err != nil {
		t.Fatal(err)
	}
	var restored ActivationPolicyV1
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.Validate() == nil {
		t.Fatal("serialized policy forged verification proof")
	}
}

func TestHardforkActivationRejectsSignedNonCanonicalReasonV1(t *testing.T) {
	_, s, b := hardforkBindingReleaseFixtureV1(t)
	d := hardforkArtifactDecisionV1(s, b)
	for _, reason := range []string{"UNKNOWN", "CHANGE_MAX_SUPPLY"} {
		d.Reasons = []string{reason}
		r := signActivationReleaseFixtureV1(t, d)
		if _, err := VerifyHardforkActivationV1(r, s, b); err == nil {
			t.Fatalf("signed forbidden/unknown reason activated: %s", reason)
		}
	}
}
