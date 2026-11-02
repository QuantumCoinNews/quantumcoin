package consensus

import "testing"

func TestNFTV4PreservesAtomV3DenominationV1(t *testing.T) {
	t.Run("monetary unit", func(t *testing.T) {
		got, err := MonetaryUnitAtProtocolV1(ProtocolNFTV4)
		if err != nil {
			t.Fatalf("NFTV4 monetary unit: %v", err)
		}
		if got != MonetaryUnitAtomV1 {
			t.Fatalf("NFTV4 monetary unit=%q, want %q", got, MonetaryUnitAtomV1)
		}
	})

	t.Run("atom denomination selector", func(t *testing.T) {
		if !ProtocolUsesAtomDenominationV1(ProtocolNFTV4) {
			t.Fatal("ProtocolUsesAtomDenominationV1(NFTV4)=false, want true")
		}
	})

	t.Run("whole QC conversion", func(t *testing.T) {
		got, err := WholeQCAmountToProtocolUnitsV1(ProtocolNFTV4, 1)
		if err != nil {
			t.Fatalf("NFTV4 one-QC conversion: %v", err)
		}
		if got != 1_000_000 {
			t.Fatalf("NFTV4 one-QC units=%d, want 1000000 atoms", got)
		}
	})
}
