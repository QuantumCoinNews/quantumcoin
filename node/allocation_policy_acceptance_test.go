package node

import (
	"math"
	"quantumcoin/economycontrol"
	"testing"
)

func TestAllocationRuntimeGateRejectsOverflowAndSemanticDrift(t *testing.T) {
	cfg := defaultRuntimePolicyConfigV1(t)
	for _, mode := range []string{"overflow", "swap", "unlock", "liquify_network", "new_mint"} {
		t.Run(mode, func(t *testing.T) {
			policies := DefaultRuntimePolicySetV1()
			a := policies.Economy.Allocations
			switch mode {
			case "overflow":
				policies.Economy.Allocations = append(a, economycontrol.Allocation{Name: "extra_a", BasisPoints: math.MaxInt}, economycontrol.Allocation{Name: "extra_b", BasisPoints: math.MaxInt}, economycontrol.Allocation{Name: "extra_c", BasisPoints: 2})
			case "swap":
				a[0].BasisPoints, a[2].BasisPoints = a[2].BasisPoints, a[0].BasisPoints
			case "unlock":
				a[1].Locked = false
			case "liquify_network":
				a[2].Liquid = true
			case "new_mint":
				a[0].MintsNewQC = true
			}
			if err := ValidateRuntimePolicyGateV1(cfg, policies); err == nil {
				t.Fatal("invalid allocation reached runtime")
			}
		})
	}
	if err := ValidateRuntimePolicyGateV1(cfg, DefaultRuntimePolicySetV1()); err != nil {
		t.Fatal(err)
	}
}
