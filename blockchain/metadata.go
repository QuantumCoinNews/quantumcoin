package blockchain

import "fmt"

type RewardBreakdown struct {
	Height        int64
	Timestamp     int64
	BaseSubsidy   int64
	AnnualBonus   int64
	FeesCollected int64

	ToMiner     int64
	ToStakers   int64
	ToDev       int64
	ToBurn      int64
	ToCommunity int64

	Total          int64
	NFTAwarded     bool
	NFTDeterminism string
}

func RewardMetadataKV(rb RewardBreakdown) map[string]string {
	m := map[string]string{
		"reward.base_subsidy": fmt.Sprintf("%d", rb.BaseSubsidy),
		"reward.annual_bonus": fmt.Sprintf("%d", rb.AnnualBonus),
		"reward.fees":         fmt.Sprintf("%d", rb.FeesCollected),
		"reward.to_miner":     fmt.Sprintf("%d", rb.ToMiner),
		"reward.to_stakers":   fmt.Sprintf("%d", rb.ToStakers),
		"reward.to_dev":       fmt.Sprintf("%d", rb.ToDev),
		"reward.to_burn":      fmt.Sprintf("%d", rb.ToBurn),
		"reward.to_community": fmt.Sprintf("%d", rb.ToCommunity),
		"reward.total":        fmt.Sprintf("%d", rb.Total),
	}
	if rb.NFTAwarded {
		m["nft.drop"] = "1"
		m["nft.det"] = rb.NFTDeterminism
	} else {
		m["nft.drop"] = "0"
	}
	return m
}
