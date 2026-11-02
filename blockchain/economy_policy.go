package blockchain

import qconsensus "quantumcoin/consensus"

// ECONOMY_POLICY_V1
// Fixed-supply pool policy for QuantumCoin.
// This file defines policy constants only. It does not mint, transfer, or alter balances.

const (
	EconomyPolicyVersion              = "ECONOMY_POLICY_V1"
	QCTotalSupplyHardCap        int64 = qconsensus.CanonicalTotalSupplyQCV1
	QCFounderTotalPool          int64 = qconsensus.CanonicalFounderTotalQCV1
	QCFounderLiquidPool         int64 = qconsensus.CanonicalFounderLiquidQCV1
	QCFounderLockedPool         int64 = qconsensus.CanonicalFounderLockedQCV1
	QCMiningEmissionPool        int64 = qconsensus.CanonicalMiningEmissionPoolQCV1
	QCMiningEmissionPoolAddress       = qconsensus.CanonicalMiningEmissionPoolAddressV1
	QCBurnSinkAddress                 = qconsensus.CanonicalBurnSinkAddressV1
)

// EconomyPolicySummary returns a read-only policy summary.
// Safe boundary: no mint, no transfer, no balance change.
func EconomyPolicySummary() map[string]any {
	return map[string]any{
		"version":            EconomyPolicyVersion,
		"totalSupplyHardCap": QCTotalSupplyHardCap,
		"founderTotalPool":   QCFounderTotalPool,
		"founderLiquidPool":  QCFounderLiquidPool,
		"founderLockedPool":  QCFounderLockedPool,
		"miningEmissionPool": QCMiningEmissionPool,
		"addresses": map[string]string{
			"miningPool": QCMiningEmissionPoolAddress,
			"burnSink":   QCBurnSinkAddress,
		},
		"safeBoundary": map[string]bool{
			"mintsNewQC":         false,
			"changesTotalSupply": false,
			"changesBalances":    false,
			"performsTransfer":   false,
		},
	}
}
