package consensus

import "testing"

func TestCanonicalEconomyHasNoReservedBonusOrNFTSupplyV1(t *testing.T) {
want := CanonicalTotalSupplyQCV1 - CanonicalFounderTotalQCV1

if CanonicalMiningEmissionPoolQCV1 != want {
t.Fatalf(
"mining emission pool must equal total supply minus founder allocation: got %d want %d",
CanonicalMiningEmissionPoolQCV1,
want,
)
}
}
