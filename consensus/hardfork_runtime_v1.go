package consensus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LoadHardforkActivationV1 receives a release verified against the compiled
// trust source. Config never chooses signers, a manifest or artifact directory.
// The executable being started is hashed, not an operator-provided binary.
func LoadHardforkActivationV1(release *HardforkReleaseV1, snapshotDir string) (ActivationPolicyV1, error) {
	var zero ActivationPolicyV1
	if release == nil {
		return zero, nil
	}
	if _, err := CanonicalHardforkDecisionPayloadV1(release.Decision); err != nil {
		return zero, err
	}
	hash := strings.ToLower(strings.TrimSpace(release.Decision.SnapshotSHA256))
	snapshotPath := filepath.Join(snapshotDir, hash+".snapshot")
	snapshot, err := os.ReadFile(snapshotPath)
	if err != nil {
		return zero, fmt.Errorf("hardfork snapshot artifact: %w", err)
	}
	executable, err := os.Executable()
	if err != nil {
		return zero, fmt.Errorf("hardfork running executable: %w", err)
	}
	binary, err := os.ReadFile(executable)
	if err != nil {
		return zero, fmt.Errorf("hardfork release binary artifact: %w", err)
	}
	return VerifyHardforkActivationV1(*release, snapshot, binary)
}

func LoadCanonicalHardforkReleaseV1() (*HardforkReleaseV1, string, error) {
	trust := CanonicalHardforkTrustV1()
	if trust == nil {
		return nil, "", nil
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, "", err
	}
	return LoadHardforkReleaseManifestV1(filepath.Dir(executable), *trust)
}

// The fixed detached manifest cannot supply or override authorities/quorums.
// A manifest is mandatory once the compiled trust root enables governance.
func LoadHardforkReleaseManifestV1(directory string, trust HardforkTrustV1) (*HardforkReleaseV1, string, error) {
	data, err := os.ReadFile(filepath.Join(directory, "hardfork-release.json"))
	if err != nil {
		return nil, "", fmt.Errorf("hardfork manifest: %w", err)
	}
	var manifest struct {
		Decision  HardforkDecisionV1
		Approvals HardforkApprovalSetV1
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return nil, "", fmt.Errorf("hardfork manifest: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, "", fmt.Errorf("hardfork manifest: trailing content")
	}
	release := &HardforkReleaseV1{Trust: trust, Decision: manifest.Decision, Approvals: manifest.Approvals}
	return release, filepath.Join(directory, "hardfork"), nil
}

// LoadCanonicalActivationPolicyV1 is the common artifact-aware resolver used
// by all canonical binders. Validation/height selection remain pure.
func LoadCanonicalActivationPolicyV1() (ActivationPolicyV1, error) {
	release, dir, err := LoadCanonicalHardforkReleaseV1()
	if err != nil {
		return ActivationPolicyV1{}, err
	}
	return LoadHardforkActivationV1(release, dir)
}
