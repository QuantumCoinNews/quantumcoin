// Package nft contains QuantumCoin project code for this module.
package nft

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

const NFTSafeBoundaryVersion = "NFT_SAFE_BOUNDARY_V1"

var (
	ErrNFTAutoMintDisabled = errors.New("nft: automatic block-based NFT mint is disabled by safe boundary")
	ErrNFTInvalidAddress   = errors.New("nft: invalid address")
	ErrNFTInvalidPolicy    = errors.New("nft: invalid policy")
)

type MonthlyRareNFTPolicy struct {
	Name              string `json:"name"`
	Enabled           bool   `json:"enabled"`
	Period            string `json:"period"`
	MaxAwardsPerMonth int    `json:"max_awards_per_month"`
	Source            string `json:"source"`
	RequiresAIReview  bool   `json:"requires_ai_review"`
	RequiresAdmin     bool   `json:"requires_admin"`
	Notes             string `json:"notes"`
}

type NFTEligibilityDecision struct {
	Eligible      bool   `json:"eligible"`
	WalletAddress string `json:"wallet_address"`
	PeriodKey     string `json:"period_key"`
	NFTType       string `json:"nft_type"`
	Reason        string `json:"reason"`
	RequiresAdmin bool   `json:"requires_admin"`
	CreatedAt     int64  `json:"created_at"`
}

func DefaultMonthlyRareNFTPolicy() MonthlyRareNFTPolicy {
	return MonthlyRareNFTPolicy{
		Name:              "QuantumCoin Telegram Game Monthly Rare NFT",
		Enabled:           true,
		Period:            "monthly",
		MaxAwardsPerMonth: 1,
		Source:            "telegram_game",
		RequiresAIReview:  true,
		RequiresAdmin:     true,
		Notes:             "Collect weekly candidates, select one monthly winner after AI risk review and admin approval.",
	}
}

func MonthKey(t time.Time) string {
	if t.IsZero() {
		t = time.Now().UTC()
	}
	u := t.UTC()
	return fmt.Sprintf("%04d-%02d", u.Year(), int(u.Month()))
}

func EvaluateMonthlyRareNFTCandidate(walletAddress string, now time.Time) (NFTEligibilityDecision, error) {
	walletAddress = strings.TrimSpace(walletAddress)
	if walletAddress == "" {
		return NFTEligibilityDecision{}, ErrNFTInvalidAddress
	}

	policy := DefaultMonthlyRareNFTPolicy()
	if !policy.Enabled || policy.MaxAwardsPerMonth != 1 {
		return NFTEligibilityDecision{}, ErrNFTInvalidPolicy
	}

	return NFTEligibilityDecision{
		Eligible:      false,
		WalletAddress: walletAddress,
		PeriodKey:     MonthKey(now),
		NFTType:       "rare",
		Reason:        "not eligible: a server-verified game session in the award period is required",
		RequiresAdmin: true,
		CreatedAt:     time.Now().Unix(),
	}, nil
}

func CheckAndMintMinerNFT(minerAddress string, blockIndex int) (string, error) {
	return "", ErrNFTAutoMintDisabled
}

func BuildDeterministicNFTID(walletAddress, nftType, periodKey string) (string, error) {
	walletAddress = strings.TrimSpace(walletAddress)
	nftType = strings.TrimSpace(nftType)
	periodKey = strings.TrimSpace(periodKey)

	if walletAddress == "" {
		return "", ErrNFTInvalidAddress
	}
	if nftType == "" {
		nftType = "rare"
	}
	if periodKey == "" {
		periodKey = MonthKey(time.Now())
	}

	base := fmt.Sprintf("QC-NFT|%s|%s|%s", walletAddress, nftType, periodKey)
	sum := sha256.Sum256([]byte(base))
	return "QC-NFT-" + hex.EncodeToString(sum[:8]), nil
}
