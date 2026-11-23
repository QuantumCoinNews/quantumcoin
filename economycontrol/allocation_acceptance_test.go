package economycontrol

import (
	"math"
	"testing"
)

func TestAllocationRejectsUnknownOverflowShares(t *testing.T) {
	p := DefaultPolicyV1()
	p.Allocations = append(p.Allocations, Allocation{Name: "extra_a", BasisPoints: math.MaxInt}, Allocation{Name: "extra_b", BasisPoints: math.MaxInt}, Allocation{Name: "extra_c", BasisPoints: 2})
	if err := ValidatePolicyV1(p); err == nil {
		t.Fatal("unknown allocations wrapped total back to 10000 and bypassed validator")
	}
}

func TestAllocationRejectsNonCanonicalNames(t *testing.T) {
	p := DefaultPolicyV1()
	p.Allocations[0].Name = " " + p.Allocations[0].Name + " "
	if err := ValidatePolicyV1(p); err == nil {
		t.Fatal("ambiguous allocation name accepted")
	}
}

func TestAllocationEveryFlagAndShareIsBoundToName(t *testing.T) {
	for index, want := range DefaultPolicyV1().Allocations {
		for bits := 0; bits < 8; bits++ {
			t.Run(want.Name+string(rune('0'+bits)), func(t *testing.T) {
				p := DefaultPolicyV1()
				a := &p.Allocations[index]
				a.Locked = bits&1 != 0
				a.Liquid = bits&2 != 0
				a.MintsNewQC = bits&4 != 0
				valid := a.Locked == want.Locked && a.Liquid == want.Liquid && !a.MintsNewQC
				if err := ValidatePolicyV1(p); (err == nil) != valid {
					t.Fatalf("flag semantics accepted=%v want=%v error=%v", err == nil, valid, err)
				}
			})
		}
		p := DefaultPolicyV1()
		other := (index + 1) % len(p.Allocations)
		p.Allocations[index].BasisPoints++
		p.Allocations[other].BasisPoints--
		if err := ValidatePolicyV1(p); err == nil {
			t.Fatal("sum-preserving share drift accepted")
		}
	}
	p := DefaultPolicyV1()
	p.Allocations[0], p.Allocations[2] = p.Allocations[2], p.Allocations[0]
	if err := ValidatePolicyV1(p); err != nil {
		t.Fatalf("order-independent valid allocation rejected: %v", err)
	}
}
