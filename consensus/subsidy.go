package consensus

import "fmt"

// LegacyMinimumSubsidyQC mevcut legacy zincirde blok ödülünün
// düşebileceği en düşük tam QC miktarıdır.
const LegacyMinimumSubsidyQC int64 = 1

// SubsidyAtHeight verilen consensus parametrelerine göre
// blok yüksekliğinin madencilik ödülünü hesaplar.
//
// Transaction ücretleri bu değere dahil değildir.
// Toplam arz sınırı coinbase doğrulama aşamasında kontrol edilir.
func SubsidyAtHeight(
	params Params,
	height uint64,
) (int64, error) {
	if err := params.Validate(); err != nil {
		return 0, fmt.Errorf(
			"%s: invalid subsidy params: %w",
			Version,
			err,
		)
	}
	effectiveHeight := height
	if effectiveHeight < 1 {
		effectiveHeight = 1
	}

	halvings := (effectiveHeight - 1) / params.HalvingIntervalBlocks
	subsidy := params.InitialSubsidyQC

	for i := uint64(0); i < halvings; i++ {
		if subsidy <= LegacyMinimumSubsidyQC {
			return LegacyMinimumSubsidyQC, nil
		}

		subsidy /= 2

		if subsidy < LegacyMinimumSubsidyQC {
			subsidy = LegacyMinimumSubsidyQC
		}
	}

	return subsidy, nil
}

// LegacySubsidyAtHeight mevcut QuantumCoin legacy
// parametreleriyle blok ödülünü hesaplar.
// CapSubsidyByRemainingSupply temel blok ödülünü kalan toplam arza göre sınırlar.
//
// totalSupply sıfır veya negatifse legacy sınırsız-arz davranışı korunur.
// Transaction ücretleri basılan yeni para olmadığı için bu hesaba dahil değildir.
func CapSubsidyByRemainingSupply(
	baseSubsidy int64,
	totalMinted int64,
	totalSupply int64,
) int64 {
	if baseSubsidy <= 0 {
		return 0
	}

	if totalSupply <= 0 {
		return baseSubsidy
	}

	remaining := totalSupply - totalMinted
	if remaining <= 0 {
		return 0
	}

	if baseSubsidy > remaining {
		return remaining
	}

	return baseSubsidy
}
func LegacySubsidyAtHeight(height uint64) (int64, error) {
	return SubsidyAtHeight(
		LegacyParamsV1(),
		height,
	)
}
