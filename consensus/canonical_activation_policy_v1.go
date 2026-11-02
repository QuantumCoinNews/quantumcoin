package consensus

// CanonicalActivationPolicyV1 returns the locally authoritative activation
// policy.
//
// Activation intentionally remains disabled until the real network ID,
// immutable legacy genesis hash and activation height are finalized.
//
// A fresh value is returned on every call, so callers cannot mutate shared
// consensus state.
func CanonicalActivationPolicyV1() ActivationPolicyV1 {
	// Artifact-less callers must fail closed if signed activation is configured.
	// An incomplete enabled policy intentionally fails Validate.
	if CanonicalHardforkTrustV1() != nil {
		return ActivationPolicyV1{Enabled: true}
	}
	return ActivationPolicyV1{}
}
