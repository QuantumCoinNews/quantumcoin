package node

import (
	"strings"
	"testing"
)

func TestQMA0065RuntimeRejectsReasonTamperingV1(t *testing.T) {
	cfg := defaultRuntimePolicyConfigV1(t)
	for _, reason := range []string{"UNKNOWN", "CHANGE_MAX_SUPPLY", "CRITICAL_CONSENSUS_BUG"} {
		t.Run(reason, func(t *testing.T) {
			policies := DefaultRuntimePolicySetV1()
			policies.Hardfork.AllowedReasons = append(policies.Hardfork.AllowedReasons, reason)
			err := ValidateRuntimePolicyGateV1(cfg, policies)
			if err == nil {
				t.Fatal("runtime accepted tampered hardfork reasons")
			}
			if !strings.Contains(err.Error(), "hardfork policy") {
				t.Fatalf("unrelated rejection: %v", err)
			}
		})
	}
}
