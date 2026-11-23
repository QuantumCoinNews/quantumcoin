package ai

// AIBonus is a retired compatibility response, never monetary state.
type AIBonus struct {
	TotalBonusQC float64 `json:"total_bonus_qc"`
	LastBonusQC  float64 `json:"last_bonus_qc"`
	Reason       string  `json:"reason"`
	UpdatedAt    int64   `json:"updated_at"`
}

// Deprecated: these methods have no state, callbacks or financial effect.
func SetBonus(totalBonus, lastBonus float64, reason string) {}
func AddBonus(delta float64, reason string)                 {}
func GetBonusSnapshot() AIBonus                             { return AIBonus{} }
