package ai

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type AIBonusFile struct {
	Timestamp int64             `json:"ts"`
	Bonuses   []BonusSuggestion `json:"ai_bonuses"`
}

// blockchain.MineBlock(...) içinden çağırdıklarımız buraya gelecek
func WriteAIBonusesTo(dir string, bonuses []BonusSuggestion) error {
	if len(bonuses) == 0 {
		return nil
	}

	payload := AIBonusFile{
		Timestamp: time.Now().Unix(),
		Bonuses:   bonuses,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(dir, "ai_bonuses.json")
	return os.WriteFile(path, data, 0o644)
}
