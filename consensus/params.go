package consensus

import (
	"fmt"
	"time"
)

// Mevcut QuantumCoin zincirinin legacy consensus değerleri.
//
// Bu adlar geriye dönük kaynak uyumluluğu için korunur.
// Ekonomik değerlerin tek kanonik kaynağı
// canonical_economy_policy_v1.go dosyasıdır.
const (
	LegacyTotalSupplyQC int64 = CanonicalTotalSupplyQCV1

	LegacyInitialSubsidyQC int64 = CanonicalInitialMiningRewardQCV1

	LegacyHalvingIntervalBlocks uint64 = 2_102_400

	LegacyTargetBlockTime time.Duration = 30 * time.Second
)

// Params bir QuantumCoin consensus sürümünün değişmez
// temel parametrelerini tanımlar.
type Params struct {
	Protocol ProtocolVersion

	TotalSupplyQC int64

	InitialSubsidyQC int64

	HalvingIntervalBlocks uint64

	TargetBlockTime time.Duration
}

// LegacyParamsV1 mevcut zincirde kullanılan consensus
// parametrelerini döndürür.
func LegacyParamsV1() Params {
	return Params{
		Protocol:              ProtocolLegacyV1,
		TotalSupplyQC:         LegacyTotalSupplyQC,
		InitialSubsidyQC:      LegacyInitialSubsidyQC,
		HalvingIntervalBlocks: LegacyHalvingIntervalBlocks,
		TargetBlockTime:       LegacyTargetBlockTime,
	}
}

// Validate consensus parametrelerinin temel olarak geçerli
// olduğunu doğrular.
func (p Params) Validate() error {
	switch p.Protocol {
	case ProtocolLegacyV1, ProtocolContinuityV2:
	default:
		return fmt.Errorf(
			"%s: unknown protocol %q",
			Version,
			p.Protocol,
		)
	}

	if p.TotalSupplyQC <= 0 {
		return fmt.Errorf(
			"%s: total supply must be greater than zero",
			Version,
		)
	}

	if p.InitialSubsidyQC <= 0 {
		return fmt.Errorf(
			"%s: initial subsidy must be greater than zero",
			Version,
		)
	}

	if p.InitialSubsidyQC > p.TotalSupplyQC {
		return fmt.Errorf(
			"%s: initial subsidy cannot exceed total supply",
			Version,
		)
	}

	if p.HalvingIntervalBlocks == 0 {
		return fmt.Errorf(
			"%s: halving interval must be greater than zero",
			Version,
		)
	}

	if p.TargetBlockTime <= 0 {
		return fmt.Errorf(
			"%s: target block time must be greater than zero",
			Version,
		)
	}

	return nil
}
