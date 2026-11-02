package node

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/consensus"
)

func hardforkStartupFixtureV1(t *testing.T) (*config.Config, consensus.HardforkReleaseV1, string) {
	t.Helper()
	cfg := defaultRuntimePolicyConfigV1(t)
	cfg.BonusFile = filepath.Join(t.TempDir(), "bonus.json")
	chain := blockchain.NewBlockchain(cfg.InitialReward, cfg.TotalSupply)
	if err := chain.SaveToFile(cfg.ChainFile); err != nil {
		t.Fatal(err)
	}
	snapshot := blockchain.SerializeBlockchain(chain)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	sh, bh := sha256.Sum256(snapshot), sha256.Sum256(binary)
	d := consensus.HardforkDecisionV1{
		NetworkID: "test-only-startup", LegacyGenesisHex: hex.EncodeToString(chain.Blocks[0].Hash),
		ActivationHeight: 10, TargetProtocol: consensus.ProtocolAtomV3,
		Reasons:        []string{"NETWORK_WIDE_PROTOCOL_UPGRADE_WITH_PUBLIC_NOTICE"},
		SnapshotSHA256: hex.EncodeToString(sh[:]), ReleaseBinarySHA256: hex.EncodeToString(bh[:]),
	}
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, d.SnapshotSHA256+".snapshot"), snapshot, 0600); err != nil {
		t.Fatal(err)
	}
	return cfg, signActivationReleaseFixtureV1(t, d), directory
}

func TestHardforkStartupEnforcesSignedHeightV1(t *testing.T) {
	cfg, r, dir := hardforkStartupFixtureV1(t)
	runtime, err := bootstrapWithHardforkReleaseV1(cfg, false, &r, dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range []int{9, 10, 11} {
		want := consensus.ProtocolLegacyV1
		if h >= 10 {
			want = consensus.ProtocolAtomV3
		}
		if got := runtime.Blockchain.ProtocolAtHeightV1(h); got != want {
			t.Fatalf("height %d: %s", h, got)
		}
	}
	p := runtime.Blockchain.ActivationPolicySnapshotV1()
	if p.Plan.ActivationHeight != r.Decision.ActivationHeight || p.Plan.NetworkID != r.Decision.NetworkID {
		t.Fatal("startup plan not derived from signed decision")
	}
	cfg.InitialReward++
	cfg.HTTPPort = "19999"
	r.Trust.Authorities.FounderQuorum = 0
	r.Decision.ActivationHeight = 1
	if got := runtime.Blockchain.ActivationPolicySnapshotV1(); got != p {
		t.Fatal("node-local mutation changed activation")
	}
	if err := runtime.Blockchain.BindActivationPolicySynchronizedV1(p); err == nil {
		t.Fatal("one-time binding was bypassed")
	}
}

func TestHardforkStartupRejectsInvalidReleaseV1(t *testing.T) {
	tests := []struct {
		name, want string
		mutate     func(*testing.T, *consensus.HardforkReleaseV1, string)
	}{
		{"height", "signature", func(t *testing.T, r *consensus.HardforkReleaseV1, _ string) { r.Decision.ActivationHeight++ }},
		{"protocol", "signature", func(t *testing.T, r *consensus.HardforkReleaseV1, _ string) {
			r.Decision.TargetProtocol = consensus.ProtocolContinuityV2
		}},
		{"network", "network mismatch", func(t *testing.T, r *consensus.HardforkReleaseV1, _ string) { r.Decision.NetworkID = "other" }},
		{"genesis", "genesis mismatch", func(t *testing.T, r *consensus.HardforkReleaseV1, _ string) {
			r.Decision.LegacyGenesisHex = strings.Repeat("ee", 32)
		}},
		{"chain_genesis", "legacy genesis hash mismatch", func(t *testing.T, r *consensus.HardforkReleaseV1, _ string) {
			d := r.Decision
			d.LegacyGenesisHex = strings.Repeat("ee", 32)
			*r = signActivationReleaseFixtureV1(t, d)
		}},
		{"snapshot", "snapshot hash mismatch", func(t *testing.T, r *consensus.HardforkReleaseV1, dir string) {
			if err := os.WriteFile(filepath.Join(dir, r.Decision.SnapshotSHA256+".snapshot"), []byte("wrong state"), 0600); err != nil {
				t.Fatal(err)
			}
		}},
		{"binary", "release binary hash mismatch", func(t *testing.T, r *consensus.HardforkReleaseV1, _ string) {
			d := r.Decision
			d.ReleaseBinarySHA256 = strings.Repeat("aa", 32)
			*r = signActivationReleaseFixtureV1(t, d)
		}},
		{"founder_quorum", "founder quorum", func(t *testing.T, r *consensus.HardforkReleaseV1, _ string) { r.Approvals.Founder = nil }},
		{"admin_quorum", "admin quorum", func(t *testing.T, r *consensus.HardforkReleaseV1, _ string) { r.Approvals.Admin = nil }},
		{"unauthorized", "unauthorized", func(t *testing.T, r *consensus.HardforkReleaseV1, _ string) {
			r.Approvals.Admin[0].PublicKey = r.Approvals.Founder[0].PublicKey
		}},
		{"duplicate_signer", "duplicate admin signer", func(t *testing.T, r *consensus.HardforkReleaseV1, _ string) {
			r.Approvals.Admin = append(r.Approvals.Admin, r.Approvals.Admin[0])
		}},
		{"duplicate_authority", "duplicate admin authority", func(t *testing.T, r *consensus.HardforkReleaseV1, _ string) {
			r.Trust.Authorities.AdminPublicKeys = append(r.Trust.Authorities.AdminPublicKeys, r.Trust.Authorities.AdminPublicKeys[0])
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, r, dir := hardforkStartupFixtureV1(t)
			tc.mutate(t, &r, dir)
			runtime, err := bootstrapWithHardforkReleaseV1(cfg, false, &r, dir)
			if err == nil || runtime != nil {
				t.Fatal("invalid release started a runtime")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("unrelated rejection: %v, want %s", err, tc.want)
			}
		})
	}
}

func TestHardforkRuntimeGateRejectsUnsignedOrMutatedPlanV1(t *testing.T) {
	cfg := defaultRuntimePolicyConfigV1(t)
	p := consensus.ActivationPolicyV1{Enabled: true, Plan: consensus.ActivationPlan{
		NetworkID: "test-only-gate", LegacyGenesisHex: strings.Repeat("aa", 32),
		ActivationHeight: 10, TargetProtocol: consensus.ProtocolAtomV3, PreserveHistory: true, PreserveBalances: true,
	}}
	if err := validateRuntimePolicyGateWithActivationV1(cfg, DefaultRuntimePolicySetV1(), p); err == nil {
		t.Fatal("gate accepted unsigned activation")
	}
	p = verifiedActivationFixtureV1(t, p)
	p.Plan.ActivationHeight++
	if err := validateRuntimePolicyGateWithActivationV1(cfg, DefaultRuntimePolicySetV1(), p); err == nil {
		t.Fatal("gate accepted mutated signed plan")
	}
}

func TestHardforkStartupNoDecisionStaysDisabledV1(t *testing.T) {
	cfg := defaultRuntimePolicyConfigV1(t)
	cfg.BonusFile = filepath.Join(t.TempDir(), "bonus.json")
	t.Setenv("QC_HARDFORK_ACTIVATION_HEIGHT", "1")
	t.Setenv("QC_HARDFORK_ADMIN_QUORUM", "0")
	t.Setenv("QC_HARDFORK_MANIFEST", "untrusted.json")
	runtime, err := BootstrapWithConfig(cfg, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := runtime.Blockchain.ProtocolAtHeightV1(1000000); got != consensus.ProtocolLegacyV1 {
		t.Fatal("local environment activated a hardfork")
	}
	if consensus.CanonicalHardforkTrustV1() != nil {
		t.Fatal("fixture unexpectedly installed production authorities")
	}
}

func TestHardforkDetachedManifestEnforcesTrustV1(t *testing.T) {
	cfg, r, snapshotDir := hardforkStartupFixtureV1(t)
	directory := t.TempDir()
	data, err := json.Marshal(struct {
		Decision  consensus.HardforkDecisionV1
		Approvals consensus.HardforkApprovalSetV1
	}{r.Decision, r.Approvals})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "hardfork-release.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, _, err := consensus.LoadHardforkReleaseManifestV1(directory, r.Trust)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bootstrapWithHardforkReleaseV1(cfg, false, loaded, snapshotDir); err != nil {
		t.Fatal(err)
	}
	forged := append([]byte(`{"Authorities":{},`), data[1:]...)
	if err := os.WriteFile(path, forged, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := consensus.LoadHardforkReleaseManifestV1(directory, r.Trust); err == nil {
		t.Fatal("manifest replaced trust root")
	}
	if err := os.WriteFile(path, append(data, []byte("{}")...), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := consensus.LoadHardforkReleaseManifestV1(directory, r.Trust); err == nil {
		t.Fatal("manifest accepted trailing document")
	}
	if _, _, err := consensus.LoadHardforkReleaseManifestV1(t.TempDir(), r.Trust); err == nil {
		t.Fatal("enabled trust accepted missing manifest")
	}
}
