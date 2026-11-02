package consensus

const CanonicalBurnLimitsVersionV1 = "QC_CANONICAL_BURN_LIMITS_V1"

const (
	CanonicalBurnPerTransactionMaxQCV1 int64 = 1_000

	CanonicalBurnOwnerLifetimeMaxQCV1 int64 = 10_000

	CanonicalBurnNetworkLifetimeMaxQCV1 int64 = 1_275_000
)

const (
	CanonicalBurnPerTransactionMaxAtomsV1 int64 = CanonicalBurnPerTransactionMaxQCV1 *
		CanonicalAtomsPerQCV1

	CanonicalBurnOwnerLifetimeMaxAtomsV1 int64 = CanonicalBurnOwnerLifetimeMaxQCV1 *
		CanonicalAtomsPerQCV1

	CanonicalBurnNetworkLifetimeMaxAtomsV1 int64 = CanonicalBurnNetworkLifetimeMaxQCV1 *
		CanonicalAtomsPerQCV1
)
