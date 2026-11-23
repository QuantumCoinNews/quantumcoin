package ai

import "errors"

var ErrAIBonusDisabled = errors.New("AI bonus export is retired: advisory-only boundary")

type AIBonusFile struct {
	Timestamp int64             `json:"ts"`
	Bonuses   []BonusSuggestion `json:"ai_bonuses"`
}

// Deprecated: no file is created or changed, including for an empty request.
func WriteAIBonusesTo(dir string, bonuses []BonusSuggestion) error { return ErrAIBonusDisabled }
