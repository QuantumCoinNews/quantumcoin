package consensus

import "testing"

func TestCanonicalBurnLimitsV1(t *testing.T) {
	const (
		wantPerTxQC           int64 = 1_000
		wantOwnerLifetimeQC   int64 = 10_000
		wantNetworkLifetimeQC int64 = 1_275_000
	)

	if CanonicalBurnPerTransactionMaxQCV1 != wantPerTxQC {
		t.Fatalf(
			"per-tx burn max=%d QC want=%d QC",
			CanonicalBurnPerTransactionMaxQCV1,
			wantPerTxQC,
		)
	}

	if CanonicalBurnOwnerLifetimeMaxQCV1 != wantOwnerLifetimeQC {
		t.Fatalf(
			"owner lifetime burn max=%d QC want=%d QC",
			CanonicalBurnOwnerLifetimeMaxQCV1,
			wantOwnerLifetimeQC,
		)
	}

	if CanonicalBurnNetworkLifetimeMaxQCV1 != wantNetworkLifetimeQC {
		t.Fatalf(
			"network lifetime burn max=%d QC want=%d QC",
			CanonicalBurnNetworkLifetimeMaxQCV1,
			wantNetworkLifetimeQC,
		)
	}

	if CanonicalBurnPerTransactionMaxAtomsV1 !=
		CanonicalBurnPerTransactionMaxQCV1*CanonicalAtomsPerQCV1 {
		t.Fatal("per-tx burn QC/atom denomination mismatch")
	}

	if CanonicalBurnOwnerLifetimeMaxAtomsV1 !=
		CanonicalBurnOwnerLifetimeMaxQCV1*CanonicalAtomsPerQCV1 {
		t.Fatal("owner lifetime burn QC/atom denomination mismatch")
	}

	if CanonicalBurnNetworkLifetimeMaxAtomsV1 !=
		CanonicalBurnNetworkLifetimeMaxQCV1*CanonicalAtomsPerQCV1 {
		t.Fatal("network lifetime burn QC/atom denomination mismatch")
	}

	if CanonicalBurnPerTransactionMaxAtomsV1 >
		CanonicalBurnOwnerLifetimeMaxAtomsV1 {
		t.Fatal("per-tx burn max exceeds owner lifetime max")
	}

	if CanonicalBurnOwnerLifetimeMaxAtomsV1 >
		CanonicalBurnNetworkLifetimeMaxAtomsV1 {
		t.Fatal("owner lifetime burn max exceeds network lifetime max")
	}

	if CanonicalBurnNetworkLifetimeMaxQCV1*20 !=
		CanonicalTotalSupplyQCV1 {
		t.Fatal("network lifetime burn max must equal 5% of hard cap")
	}
}
