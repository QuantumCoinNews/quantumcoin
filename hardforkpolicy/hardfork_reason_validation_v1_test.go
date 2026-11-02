package hardforkpolicy

import "testing"

func TestQMA0065RejectUnknownReasonV1(t *testing.T) {
	for _, reason := range []string{"UNKNOWN", "critical_consensus_bug", "CRITICAL_CONSENSUS_BUG "} {
		t.Run(reason, func(t *testing.T) {
			p := DefaultPolicyV1()
			p.AllowedReasons[0] = reason
			if err := ValidatePolicyV1(p); err == nil {
				t.Fatalf("accepted non-canonical allowed reason %q", reason)
			}
		})
	}
}
func TestQMA0065RejectDuplicateReasonV1(t *testing.T) {
	p := DefaultPolicyV1()
	p.AllowedReasons[1] = p.AllowedReasons[0]
	if err := ValidatePolicyV1(p); err == nil {
		t.Fatal("accepted duplicate allowed reason and missing canonical reason")
	}
}
func TestQMA0065RejectConflictingReasonV1(t *testing.T) {
	p := DefaultPolicyV1()
	p.AllowedReasons = append(p.AllowedReasons, p.ForbiddenReasons[0])
	if err := ValidatePolicyV1(p); err == nil {
		t.Fatal("accepted CHANGE_MAX_SUPPLY as both allowed and forbidden")
	}
}
func TestQMA0065RejectForbiddenReasonTamperingV1(t *testing.T) {
	for _, mutation := range []string{"unknown", "duplicate", "extra", "overlap"} {
		t.Run(mutation, func(t *testing.T) {
			p := DefaultPolicyV1()
			switch mutation {
			case "unknown":
				p.ForbiddenReasons[0] = "UNKNOWN"
			case "duplicate":
				p.ForbiddenReasons[1] = p.ForbiddenReasons[0]
			case "extra":
				p.ForbiddenReasons = append(p.ForbiddenReasons, "UNKNOWN")
			case "overlap":
				p.ForbiddenReasons[0] = p.AllowedReasons[0]
			}
			if err := ValidatePolicyV1(p); err == nil {
				t.Fatal("accepted tampered forbidden reason set")
			}
		})
	}
}
func TestQMA0065ReasonOrderDoesNotChangePolicyV1(t *testing.T) {
	p := DefaultPolicyV1()
	p.AllowedReasons[0], p.AllowedReasons[1] = p.AllowedReasons[1], p.AllowedReasons[0]
	p.ForbiddenReasons[0], p.ForbiddenReasons[1] = p.ForbiddenReasons[1], p.ForbiddenReasons[0]
	if err := ValidatePolicyV1(p); err != nil {
		t.Fatalf("same canonical sets in different order: %v", err)
	}
}
