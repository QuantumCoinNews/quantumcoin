package consensus

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const MonthlyRareNFTAwardCountV1 = 12

var ErrMonthlyNFTGameActivationRequiredV1 = errors.New("nft: Telegram Game activation time required")

// MonthlyNFTAwardPeriodsFromGameActivationV1 preserves the existing twelve
// complete UTC months following activation. The date must be supplied by
// trusted game configuration; this function never chooses a launch date.
func MonthlyNFTAwardPeriodsFromGameActivationV1(activatedAt time.Time) ([]string, error) {
	if activatedAt.IsZero() {
		return nil, ErrMonthlyNFTGameActivationRequiredV1
	}
	activation := activatedAt.UTC()
	first := time.Date(activation.Year(), activation.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	if first.Year() < 1 || first.AddDate(0, MonthlyRareNFTAwardCountV1-1, 0).Year() > 9999 {
		return nil, fmt.Errorf("monthly NFT program: activation outside supported period range")
	}
	result := make([]string, MonthlyRareNFTAwardCountV1)
	for i := range result {
		result[i] = first.AddDate(0, i, 0).Format("2006-01")
	}
	return result, nil
}

type NFTMonthlyProgramPolicyV1 struct {
	NetworkID        string
	CampaignID       string
	FirstAwardPeriod string
}

func validNFTMonthlyIdentifierV1(value string) bool {
	if value == "" || len(value) > 128 || strings.TrimSpace(value) != value {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == ':' {
			continue
		}
		return false
	}
	return true
}

func (p NFTMonthlyProgramPolicyV1) Validate() error {
	if !validNFTMonthlyIdentifierV1(p.NetworkID) || !validNFTMonthlyIdentifierV1(p.CampaignID) {
		return fmt.Errorf("monthly NFT program: invalid network/campaign")
	}
	first, err := time.Parse("2006-01", p.FirstAwardPeriod)
	if err != nil || first.Year() < 1 || first.AddDate(0, MonthlyRareNFTAwardCountV1-1, 0).Year() > 9999 {
		return fmt.Errorf("monthly NFT program: invalid first award period")
	}
	return nil
}

// Serial is the one-based month position: month one has serial 1 and month
// twelve serial 12. A thirteenth month is not part of this program.
func (p NFTMonthlyProgramPolicyV1) SerialForAwardPeriodV1(period string) (uint32, error) {
	if err := p.Validate(); err != nil {
		return 0, err
	}
	first, _ := time.Parse("2006-01", p.FirstAwardPeriod)
	month, err := time.Parse("2006-01", period)
	if err != nil {
		return 0, fmt.Errorf("monthly NFT program: invalid award period")
	}
	offset := (month.Year()-first.Year())*12 + int(month.Month()-first.Month())
	if offset < 0 || offset >= MonthlyRareNFTAwardCountV1 {
		return 0, fmt.Errorf("monthly NFT program: award outside twelve periods")
	}
	return uint32(offset + 1), nil
}

func NewNFTMonthlyProgramPolicyFromGameActivationV1(network, campaign string, activatedAt time.Time) (NFTMonthlyProgramPolicyV1, error) {
	periods, err := MonthlyNFTAwardPeriodsFromGameActivationV1(activatedAt)
	if err != nil {
		return NFTMonthlyProgramPolicyV1{}, err
	}
	p := NFTMonthlyProgramPolicyV1{NetworkID: network, CampaignID: campaign, FirstAwardPeriod: periods[0]}
	if err := p.Validate(); err != nil {
		return NFTMonthlyProgramPolicyV1{}, err
	}
	return p, nil
}
