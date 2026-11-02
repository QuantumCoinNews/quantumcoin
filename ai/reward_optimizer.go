package ai

// OptimizeRewards documents the QuantumCoin component used by this module.
// AI_SAFE_BOUNDARY_V1
// OptimizeRewards is kept only for API/backward compatibility.
// AI must not decide or create rewards. Reward decisions must remain policy/admin/pool based.
func OptimizeRewards(txs []TxLite, anoms []AnomalyReport) []BonusSuggestion {
	return nil
}

// DistributeAIBonusesLite is kept only for backward compatibility.
// It intentionally returns nil under the safe AI boundary.
func DistributeAIBonusesLite(txs []TxLite) []BonusSuggestion {
	return nil
}
