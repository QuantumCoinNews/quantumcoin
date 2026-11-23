package config

import "fmt"

// ValidateRewardRolesV1 checks bounded values before adding them and accepts
// only the implemented coinbase contract. Proposal fields cannot activate a split.
func ValidateRewardRolesV1(c *Config) error {
	if c == nil {
		return fmt.Errorf("reward configuration is missing")
	}
	total := 0
	for _, pct := range []int{c.RewardPctMiner, c.RewardPctStake, c.RewardPctDev, c.RewardPctBurn} {
		if pct < 0 || pct > 100 {
			return fmt.Errorf("each reward percentage must be 0..100")
		}
		total += pct
	}
	if total != 100 {
		return fmt.Errorf("reward percentages must total exactly 100, got %d", total)
	}
	if c.RewardPctMiner != ShareMiningPct || c.RewardPctStake != ShareStakingPct || c.RewardPctDev != ShareDevPct || c.RewardPctBurn != ShareBurnPct {
		return fmt.Errorf("unsupported reward split: active consensus requires miner=100, stake/dev/burn/community=0; legacy proposals cannot activate consensus")
	}
	return nil
}
