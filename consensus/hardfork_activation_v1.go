package consensus

import (
	"fmt"
	"strings"

	"quantumcoin/hardforkpolicy"
)

// HardforkTrustV1 is a release's reviewed trust root, never node-local config.
type HardforkTrustV1 struct {
	NetworkID        string
	LegacyGenesisHex string
	Authorities      HardforkAuthoritySetV1
}

type HardforkReleaseV1 struct {
	Trust     HardforkTrustV1
	Decision  HardforkDecisionV1
	Approvals HardforkApprovalSetV1
}

// CanonicalHardforkTrustV1 is the sole production trust source. There is no
// mutable global, environment override or config-file authority loader.
// No production authority set has been finalized; nil disables activation.
// Decisions/signatures belong in a detached manifest, since embedding a
// signature of the running binary's hash inside that binary is self-referential.
func CanonicalHardforkTrustV1() *HardforkTrustV1 {
	return nil
}

// VerifyHardforkActivationV1 derives the existing activation policy only after
// checking the trusted network, signatures, both quorums and actual artifacts.
// Callers supply a trusted release, not authority keys from an untrusted peer.
// The returned value retains no caller-owned slices or artifact buffers.
func VerifyHardforkActivationV1(release HardforkReleaseV1, snapshot, releaseBinary []byte) (ActivationPolicyV1, error) {
	var zero ActivationPolicyV1
	d := release.Decision
	if err := validateHardforkSHA256HexV1("trusted genesis", release.Trust.LegacyGenesisHex); err != nil {
		return zero, err
	}
	network := strings.TrimSpace(release.Trust.NetworkID)
	genesis := strings.ToLower(strings.TrimSpace(release.Trust.LegacyGenesisHex))
	if network == "" || strings.TrimSpace(d.NetworkID) != network {
		return zero, fmt.Errorf("hardfork activation: trusted network mismatch")
	}
	if strings.ToLower(strings.TrimSpace(d.LegacyGenesisHex)) != genesis {
		return zero, fmt.Errorf("hardfork activation: trusted genesis mismatch")
	}
	allowed := make(map[string]bool)
	for _, reason := range hardforkpolicy.DefaultAllowedReasonsV1() {
		allowed[reason] = true
	}
	for _, reason := range d.Reasons {
		if !allowed[reason] {
			return zero, fmt.Errorf("hardfork activation: non-canonical reason %q", reason)
		}
	}
	if err := ValidateHardforkApprovalsV1(d, release.Trust.Authorities, release.Approvals); err != nil {
		return zero, fmt.Errorf("hardfork activation: approvals: %w", err)
	}
	if err := ValidateHardforkArtifactsV1(d, snapshot, releaseBinary); err != nil {
		return zero, fmt.Errorf("hardfork activation: artifacts: %w", err)
	}
	plan := activationPlanFromHardforkDecisionV1(d)
	p := ActivationPolicyV1{Enabled: true, Plan: plan, verified: true, verifiedPlan: plan}
	if err := p.Validate(); err != nil {
		return zero, err
	}
	return p, nil
}

func activationPlanFromHardforkDecisionV1(d HardforkDecisionV1) ActivationPlan {
	return ActivationPlan{
		NetworkID:        strings.TrimSpace(d.NetworkID),
		LegacyGenesisHex: strings.ToLower(strings.TrimSpace(d.LegacyGenesisHex)),
		ActivationHeight: d.ActivationHeight,
		TargetProtocol:   d.TargetProtocol,
		PreserveHistory:  true,
		PreserveBalances: true,
	}
}
