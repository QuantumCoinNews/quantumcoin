package nft

import (
	qconsensus "quantumcoin/consensus"
	"time"
)

// Compatibility name retained; activation refers to Telegram Game, not mainnet.
var ErrMonthlyNFTActivationRequiredV1 = qconsensus.ErrMonthlyNFTGameActivationRequiredV1

// MonthlyNFTPeriodsFromActivationV1 delegates to the one canonical program
// period generator. It preserves the twelve complete UTC month calculation.
func MonthlyNFTPeriodsFromActivationV1(activatedAt time.Time) ([]string, error) {
	return qconsensus.MonthlyNFTAwardPeriodsFromGameActivationV1(activatedAt)
}
