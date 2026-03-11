// ai/collector.go
package ai

// Bu dosya yalnızca AI'nin önerdiği bonusları diske yazdırmak için
// üstten çağrılabilecek küçük bir sarmalayıcı tutuyor.
// Asıl işi export.go'daki WriteAIBonusesTo yapıyor.

func CollectAndWrite(dir string, bonuses []BonusSuggestion) error {
	return WriteAIBonusesTo(dir, bonuses)
}
