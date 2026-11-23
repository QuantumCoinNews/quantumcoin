package consensus

import "testing"

func TestCanonicalEmissionProjectionV1(t *testing.T) {
	projection, ok := ProjectCanonicalEmissionV1(CanonicalFounderTotalQCV1 + CanonicalInitialMiningRewardQCV1)
	if !ok || projection.LastSubsidyHeight != 484499 || projection.LastSubsidyQC != 50 ||
		projection.TargetSeconds != 14534970 || projection.LastSubsidyHeight >= projection.FirstHalvingHeight {
		t.Fatalf("unexpected projection: %+v", projection)
	}
	partial, ok := ProjectCanonicalEmissionV1(CanonicalTotalSupplyQCV1 - 51)
	if !ok || partial.LastSubsidyHeight != 2 || partial.LastSubsidyQC != 1 {
		t.Fatalf("partial reward was not capped: %+v", partial)
	}
	if _, ok := ProjectCanonicalEmissionV1(-1); ok {
		t.Fatal("negative genesis issuance accepted")
	}
}
