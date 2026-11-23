package game

// Historical thresholds are not economic policy or payable milestones.
const (
	RewardThresholdBronze = 500
	RewardThresholdSilver = 1000
	RewardThresholdGold   = 2000
)

// GiveReward never prints a fake payment or changes a balance.
// Deprecated: rewards require canonical Telegram evidence and ledger accounting.
func GiveReward(player string, rewardType string, amount int) error { return ErrLegacyGameDisabled }

// EvaluateAndReward explicitly rejects the retired demo milestone path.
// No claim state is fabricated for rewards that do not exist.
func EvaluateAndReward(gs *GameState, player string) error { return ErrLegacyGameDisabled }
