package node

import (
	"reflect"
	"sync"
	"testing"

	"quantumcoin/burncontrol"
	"quantumcoin/economycontrol"
	"quantumcoin/foundervesting"
	"quantumcoin/hardforkpolicy"
	"quantumcoin/miningtransition"
	"quantumcoin/networkfee"
	"quantumcoin/persistence"
	"quantumcoin/queueworker"
)

type safeBoundaryCaseV1 struct {
	name     string
	boundary map[string]bool
	validate func() error
}

func freshSafeBoundaryCasesV1() []safeBoundaryCaseV1 {
	b := burncontrol.DefaultPolicyV1()
	e := economycontrol.DefaultPolicyV1()
	f := foundervesting.DefaultPolicyV1()
	h := hardforkpolicy.DefaultPolicyV1()
	n := networkfee.DefaultPolicyV1()
	m := miningtransition.DefaultPolicyV1()
	p := persistence.DefaultPlanV1()
	q := queueworker.DefaultPlanV1()
	r := queueworker.RuntimePlanV1()
	return []safeBoundaryCaseV1{
		{"burn", b.SafeBoundary, func() error { return burncontrol.ValidatePolicyV1(b) }},
		{"economy", e.SafeBoundary, func() error { return economycontrol.ValidatePolicyV1(e) }},
		{"founder", f.SafeBoundary, func() error { return foundervesting.ValidatePolicyV1(f) }},
		{"hardfork", h.SafeBoundary, func() error { return hardforkpolicy.ValidatePolicyV1(h) }},
		{"fee", n.SafeBoundary, func() error { return networkfee.ValidatePolicyV1(n) }},
		{"transition", m.SafeBoundary, func() error { return miningtransition.ValidatePolicyV1(m) }},
		{"persistence", p.SafeBoundary, func() error { return persistence.ValidatePlanV1(p) }},
		{"queue-plan", q.SafeBoundary, func() error { return queueworker.ValidatePlanV1(q) }},
		{"queue-runtime", r.SafeBoundary, func() error { return queueworker.ValidatePlanV1(r) }},
	}
}

func cloneSafeBoundaryTestV1(m map[string]bool) map[string]bool {
	c := make(map[string]bool, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

func TestSafeBoundaryEveryKeyIsValidated(t *testing.T) {
	for index, control := range freshSafeBoundaryCasesV1() {
		if err := control.validate(); err != nil {
			t.Fatalf("positive control %s: %v", control.name, err)
		}
		for key := range control.boundary {
			for _, mode := range []string{"flip", "delete", "replace"} {
				t.Run(control.name+"/"+key+"/"+mode, func(t *testing.T) {
					c := freshSafeBoundaryCasesV1()[index]
					before := cloneSafeBoundaryTestV1(c.boundary)
					switch mode {
					case "flip":
						c.boundary[key] = !c.boundary[key]
					case "delete":
						delete(c.boundary, key)
					case "replace":
						delete(c.boundary, key)
						c.boundary["unrecognized_key"] = true
					}
					mutated := cloneSafeBoundaryTestV1(c.boundary)
					if err := c.validate(); err == nil {
						t.Fatal("invalid declaration accepted")
					}
					if !reflect.DeepEqual(mutated, c.boundary) {
						t.Fatal("validator rewrote caller declaration")
					}
					fresh := freshSafeBoundaryCasesV1()[index]
					if !reflect.DeepEqual(before, fresh.boundary) {
						t.Fatal("caller mutation changed canonical default")
					}
				})
			}
		}
		t.Run(control.name+"/extra", func(t *testing.T) {
			c := freshSafeBoundaryCasesV1()[index]
			c.boundary["unrecognized_key"] = true
			if err := c.validate(); err == nil {
				t.Fatal("unknown key accepted")
			}
		})
	}
}

func TestSafeBoundaryConcurrentIndependentDeclarations(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, c := range freshSafeBoundaryCasesV1() {
				if err := c.validate(); err != nil {
					t.Errorf("fresh %s: %v", c.name, err)
				}
				for key := range c.boundary {
					c.boundary[key] = !c.boundary[key]
				}
			}
		}()
	}
	wg.Wait()
	for _, c := range freshSafeBoundaryCasesV1() {
		if err := c.validate(); err != nil {
			t.Fatalf("global declaration changed: %s: %v", c.name, err)
		}
	}
}

func TestSafeBoundaryRuntimeGateRejectsEveryCorruptedKey(t *testing.T) {
	cfg := defaultRuntimePolicyConfigV1(t)
	selectMaps := func(p RuntimePolicySetV1) []map[string]bool {
		return []map[string]bool{p.Burn.SafeBoundary, p.Economy.SafeBoundary, p.FounderVesting.SafeBoundary, p.Hardfork.SafeBoundary, p.NetworkFee.SafeBoundary}
	}
	for index, m := range selectMaps(DefaultRuntimePolicySetV1()) {
		for key := range m {
			policies := DefaultRuntimePolicySetV1()
			selectMaps(policies)[index][key] = !m[key]
			if err := ValidateRuntimePolicyGateV1(cfg, policies); err == nil {
				t.Errorf("runtime accepted package %d key %s", index, key)
			}
		}
	}
	if err := ValidateRuntimePolicyGateV1(cfg, DefaultRuntimePolicySetV1()); err != nil {
		t.Fatal(err)
	}
}
