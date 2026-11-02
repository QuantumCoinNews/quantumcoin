// Package nft contains QuantumCoin project code for this module.
package nft

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const MonthlyNFTPlaceholderVersionV1 = "NFT_MONTHLY_PLACEHOLDER_V1"

const MonthlyNFTMainnetMintApprovalV1 = "I_APPROVE_MONTHLY_NFT_MAINNET_MINT"

var (
	ErrMonthlyNFTInvalidPlan      = errors.New("nft: invalid monthly nft plan")
	ErrMonthlyNFTMintNotAllowed   = errors.New("nft: monthly nft mint is not allowed by placeholder safe boundary")
	ErrMonthlyNFTNoCandidates     = errors.New("nft: no monthly nft candidates")
	ErrMonthlyNFTInvalidCandidate = errors.New("nft: invalid monthly nft candidate")
	ErrMonthlyNFTInvalidMainnet   = errors.New("nft: mainnet monthly nft mint requires explicit approval")
)

type MonthlyNFTSlotV1 struct {
	ID                    string `json:"id"`
	Period                string `json:"period"`
	MonthIndex            int    `json:"monthIndex"`
	Name                  string `json:"name"`
	Rarity                string `json:"rarity"`
	Concept               string `json:"concept"`
	ImagePath             string `json:"imagePath"`
	MetadataPath          string `json:"metadataPath"`
	PreparedBeforeMainnet bool   `json:"preparedBeforeMainnet"`
	MintEnabled           bool   `json:"mintEnabled"`
	Minted                bool   `json:"minted"`
}

type MonthlyNFTPlanV1 struct {
	Version             string             `json:"version"`
	Year                int                `json:"year"`
	TotalSlots          int                `json:"totalSlots"`
	AwardsPerMonth      int                `json:"awardsPerMonth"`
	Rarity              string             `json:"rarity"`
	Mode                string             `json:"mode"`
	MainnetMintApproval string             `json:"mainnetMintApproval"`
	SafeBoundary        map[string]bool    `json:"safeBoundary"`
	Slots               []MonthlyNFTSlotV1 `json:"slots"`
}

type MonthlyNFTCandidateV1 struct {
	UserID        string `json:"userId"`
	WalletAddress string `json:"walletAddress"`
	Period        string `json:"period"`
	Eligible      bool   `json:"eligible"`
	AIApproved    bool   `json:"aiApproved"`
	AdminApproved bool   `json:"adminApproved"`
}

type MonthlyNFTWinnerDecisionV1 struct {
	Version       string `json:"version"`
	Period        string `json:"period"`
	WinnerUserID  string `json:"winnerUserId"`
	WalletAddress string `json:"walletAddress"`
	NFTID         string `json:"nftId"`
	SelectionHash string `json:"selectionHash"`
	MintAllowed   bool   `json:"mintAllowed"`
	Minted        bool   `json:"minted"`
	Reason        string `json:"reason"`
}

func DefaultMonthlyNFTSafeBoundaryV1() map[string]bool {
	return map[string]bool{
		"testnetPlaceholderOnly":             true,
		"mainnetMintDisabledByDefault":       true,
		"requiresAIReview":                   true,
		"requiresAdminApproval":              true,
		"requiresFinalMetadataBeforeMainnet": true,
		"requiresUniqueMonthlyNFTs":          true,
		"maxOneAwardPerMonth":                true,
		"doesNotChangeQCSupply":              true,
		"doesNotChangeQCBalance":             true,
		"doesNotMintInTestnet":               true,
	}
}

func BuildMonthlyRareNFTPlaceholderPlanV1(year int) MonthlyNFTPlanV1 {
	if year <= 0 {
		year = 2026
	}

	templates, err := monthlyNFTTemplateSlotsV1(monthlyNFTStock2026SourceV1)
	if err != nil {
		// An invalid embedded stock can never become an apparently valid plan.
		return MonthlyNFTPlanV1{}
	}
	slots := make([]MonthlyNFTSlotV1, 0, len(templates))
	for _, template := range templates {
		slot := template
		slot.Period = fmt.Sprintf("%04d-%02d", year, template.MonthIndex)
		slot.ID = fmt.Sprintf("rare-%s-001", slot.Period)
		slot.ImagePath = strings.Replace(template.ImagePath, template.ID, slot.ID, 1)
		slot.MetadataPath = strings.Replace(template.MetadataPath, template.ID, slot.ID, 1)
		slots = append(slots, slot)
	}

	return MonthlyNFTPlanV1{
		Version:             MonthlyNFTPlaceholderVersionV1,
		Year:                year,
		TotalSlots:          12,
		AwardsPerMonth:      1,
		Rarity:              "rare",
		Mode:                "testnet_placeholder",
		MainnetMintApproval: "",
		SafeBoundary:        DefaultMonthlyNFTSafeBoundaryV1(),
		Slots:               slots,
	}
}

func ValidateMonthlyNFTPlanV1(plan MonthlyNFTPlanV1) error {
	if plan.Version != MonthlyNFTPlaceholderVersionV1 {
		return fmt.Errorf("%w: invalid version", ErrMonthlyNFTInvalidPlan)
	}
	if plan.TotalSlots != 12 || len(plan.Slots) != 12 {
		return fmt.Errorf("%w: plan must contain 12 monthly slots", ErrMonthlyNFTInvalidPlan)
	}
	if plan.AwardsPerMonth != 1 {
		return fmt.Errorf("%w: awards per month must be 1", ErrMonthlyNFTInvalidPlan)
	}

	seenPeriod := map[string]bool{}
	seenID := map[string]bool{}
	seenName := map[string]bool{}
	seenConcept := map[string]bool{}
	seenImage := map[string]bool{}
	seenMetadata := map[string]bool{}

	for _, s := range plan.Slots {
		if strings.TrimSpace(s.ID) == "" || strings.TrimSpace(s.Period) == "" {
			return fmt.Errorf("%w: slot id/period required", ErrMonthlyNFTInvalidPlan)
		}
		if s.Rarity != "rare" {
			return fmt.Errorf("%w: all monthly NFTs must be rare", ErrMonthlyNFTInvalidPlan)
		}
		if s.MintEnabled || s.Minted {
			return fmt.Errorf("%w: mint must stay disabled in placeholder plan", ErrMonthlyNFTMintNotAllowed)
		}
		if seenPeriod[s.Period] || seenID[s.ID] || seenName[s.Name] || seenConcept[s.Concept] || seenImage[s.ImagePath] || seenMetadata[s.MetadataPath] {
			return fmt.Errorf("%w: all 12 NFTs must be independent and unique", ErrMonthlyNFTInvalidPlan)
		}
		seenPeriod[s.Period] = true
		seenID[s.ID] = true
		seenName[s.Name] = true
		seenConcept[s.Concept] = true
		seenImage[s.ImagePath] = true
		seenMetadata[s.MetadataPath] = true
	}

	return nil
}

func ValidateMonthlyNFTMainnetMintGateV1(plan MonthlyNFTPlanV1) error {
	if strings.EqualFold(plan.Mode, "mainnet") || strings.EqualFold(plan.Mode, "final") {
		if plan.MainnetMintApproval != MonthlyNFTMainnetMintApprovalV1 {
			return ErrMonthlyNFTInvalidMainnet
		}
		// Deprecated preview model: a public token cannot grant mint authority.
		// The runtime bridge verifies draw, final content/role approvals, signed
		// award and node-local consensus policy before confirming a mint.
		return ErrMonthlyNFTMintNotAllowed
	}
	return ValidateMonthlyNFTPlanV1(plan)
}

func SelectMonthlyNFTWinnerPlaceholderV1(period string, seed string, candidates []MonthlyNFTCandidateV1) (MonthlyNFTWinnerDecisionV1, error) {
	period = strings.TrimSpace(period)
	seed = strings.TrimSpace(seed)
	if period == "" {
		return MonthlyNFTWinnerDecisionV1{}, ErrMonthlyNFTInvalidCandidate
	}
	if len(candidates) == 0 {
		return MonthlyNFTWinnerDecisionV1{}, ErrMonthlyNFTNoCandidates
	}

	eligible := make([]MonthlyNFTCandidateV1, 0)
	seenUsers, seenWallets := map[string]bool{}, map[string]bool{}
	for _, c := range candidates {
		c.UserID = strings.TrimSpace(c.UserID)
		c.WalletAddress = strings.TrimSpace(c.WalletAddress)
		if c.UserID == "" || c.WalletAddress == "" {
			continue
		}
		if c.Period == period && c.Eligible && c.AIApproved && c.AdminApproved {
			if seenUsers[c.UserID] || seenWallets[c.WalletAddress] {
				return MonthlyNFTWinnerDecisionV1{}, fmt.Errorf("%w: duplicate eligible user or wallet", ErrMonthlyNFTInvalidCandidate)
			}
			seenUsers[c.UserID], seenWallets[c.WalletAddress] = true, true
			eligible = append(eligible, c)
		}
	}

	if len(eligible) == 0 {
		return MonthlyNFTWinnerDecisionV1{}, ErrMonthlyNFTNoCandidates
	}

	sort.SliceStable(eligible, func(i, j int) bool {
		return eligible[i].UserID < eligible[j].UserID
	})

	if seed == "" {
		seed = "QC_MONTHLY_NFT_PLACEHOLDER_SEED"
	}

	base := period + "|" + seed
	for _, c := range eligible {
		base += "|" + c.UserID + "|" + c.WalletAddress
	}

	sum := sha256.Sum256([]byte(base))
	idx := int(sum[0]) % len(eligible)
	winner := eligible[idx]

	nftID, err := BuildDeterministicNFTID(winner.WalletAddress, "rare", period)
	if err != nil {
		return MonthlyNFTWinnerDecisionV1{}, err
	}

	return MonthlyNFTWinnerDecisionV1{
		Version:       MonthlyNFTPlaceholderVersionV1,
		Period:        period,
		WinnerUserID:  winner.UserID,
		WalletAddress: winner.WalletAddress,
		NFTID:         nftID,
		SelectionHash: hex.EncodeToString(sum[:]),
		MintAllowed:   false,
		Minted:        false,
		Reason:        "placeholder winner selection only; mainnet mint requires final metadata, AI/admin approval, and explicit mainnet approval",
	}, nil
}
