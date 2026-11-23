// ai/collector.go

// Package ai contains QuantumCoin project code for this module.
package ai

// Deprecated: retained for callers of the retired AI bonus exporter.
// Always returns ErrAIBonusDisabled without changing files or balances.
func CollectAndWrite(dir string, bonuses []BonusSuggestion) error {
	return WriteAIBonusesTo(dir, bonuses)
}
