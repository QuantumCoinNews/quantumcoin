package burncontrol

import "testing"

func TestPermanentBurnCannotBeChangedIntoRecycling(t *testing.T) {
	p := DefaultPolicyV1()
	if err := ValidatePolicyV1(p); err != nil {
		t.Fatal(err)
	}
	if p.Mechanism != "PERMANENT_BURN" || p.RecyclingReserve.Enabled {
		t.Fatal("ambiguous burn mechanism")
	}
	mutations := []func(*BurnPolicy){
		func(p *BurnPolicy) { p.Mechanism = "RECYCLING_RESERVE_PROPOSAL" },
		func(p *BurnPolicy) { p.RecyclingReserve.Enabled = true },
		func(p *BurnPolicy) { p.RecyclingReserve.CountsAsBurnedSupply = true },
		func(p *BurnPolicy) { p.RecyclingReserve.MayMintReplacement = true },
		func(p *BurnPolicy) { p.RecyclingReserve.MaySpendBurnSink = true },
		func(p *BurnPolicy) { p.BurnDoesNotMintReplacement = false },
	}
	for _, mutate := range mutations {
		candidate := DefaultPolicyV1()
		mutate(&candidate)
		if err := ValidatePolicyV1(candidate); err == nil {
			t.Fatal("unsafe burn/recycle policy accepted", candidate)
		}
	}
}
