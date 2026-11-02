package consensus

import "fmt"

const QCActivationPolicyVersionV1 = "QC_ACTIVATION_POLICY_V1"

// ActivationPolicyV1 keeps protocol activation disabled until a complete,
// validated and immutable activation plan is explicitly supplied.
//
// Its zero value is intentionally safe:
// activation is disabled and all heights use the legacy protocol.
type ActivationPolicyV1 struct {
	Enabled bool
	Plan    ActivationPlan

	// Private value-only proof; serialization cannot manufacture it.
	verified     bool
	verifiedPlan ActivationPlan
}

// Validate rejects incomplete enabled policies and hidden plans placed inside
// a disabled policy.
func (p ActivationPolicyV1) Validate() error {
	if !p.Enabled {
		if p.Plan != (ActivationPlan{}) || p.verified || p.verifiedPlan != (ActivationPlan{}) {
			return fmt.Errorf(
				"%s: disabled policy must not contain an activation plan",
				QCActivationPolicyVersionV1,
			)
		}

		return nil
	}

	if err := p.Plan.Validate(); err != nil {
		return fmt.Errorf(
			"%s: activation plan: %w",
			QCActivationPolicyVersionV1,
			err,
		)
	}

	if !p.verified || p.Plan != p.verifiedPlan {
		return fmt.Errorf("hardfork activation: verified signed decision binding required")
	}

	return nil
}

// ProtocolAtHeight returns the legacy protocol unless activation is enabled
// with a completely valid plan.
//
// Invalid policy state fails closed to the legacy protocol.
func (p ActivationPolicyV1) ProtocolAtHeight(
	height int,
) ProtocolVersion {
	if !p.Enabled {
		return ProtocolLegacyV1
	}

	if err := p.Validate(); err != nil {
		return ProtocolLegacyV1
	}

	return ProtocolAtHeightTargetV1(
		height,
		p.Plan.ActivationHeight,
		p.Plan.EffectiveTargetProtocol(),
	)
}
