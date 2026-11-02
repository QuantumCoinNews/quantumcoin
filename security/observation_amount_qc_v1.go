package security

// observationAmountQCV1 preserves legacy whole-QC observations and uses the
// explicit QC amount when the producer supplies a fractional amount.
// This is advisory analysis only; it never participates in consensus.
func observationAmountQCV1(legacyAmount int, amountQC float64) float64 {
	if amountQC != 0 {
		return amountQC
	}
	return float64(legacyAmount)
}
