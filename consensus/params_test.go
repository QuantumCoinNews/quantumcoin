package consensus

import (
	"testing"
	"time"
)

func TestLegacyParamsV1(t *testing.T) {
	params := LegacyParamsV1()

	if params.Protocol != ProtocolLegacyV1 {
		t.Fatalf(
			"protocol = %q, want %q",
			params.Protocol,
			ProtocolLegacyV1,
		)
	}

	if params.TotalSupplyQC != 25_500_000 {
		t.Fatalf(
			"total supply = %d, want %d",
			params.TotalSupplyQC,
			int64(25_500_000),
		)
	}

	if params.InitialSubsidyQC != 50 {
		t.Fatalf(
			"initial subsidy = %d, want %d",
			params.InitialSubsidyQC,
			int64(50),
		)
	}

	if params.HalvingIntervalBlocks != 2_102_400 {
		t.Fatalf(
			"halving interval = %d, want %d",
			params.HalvingIntervalBlocks,
			uint64(2_102_400),
		)
	}

	if params.TargetBlockTime != 30*time.Second {
		t.Fatalf(
			"target block time = %s, want %s",
			params.TargetBlockTime,
			30*time.Second,
		)
	}

	if err := params.Validate(); err != nil {
		t.Fatalf("legacy params rejected: %v", err)
	}
}

func TestParamsValidateRejectsUnknownProtocol(t *testing.T) {
	params := LegacyParamsV1()
	params.Protocol = ProtocolVersion("UNKNOWN")

	if err := params.Validate(); err == nil {
		t.Fatal("unknown protocol must be rejected")
	}
}

func TestParamsValidateRejectsInvalidSupply(t *testing.T) {
	params := LegacyParamsV1()
	params.TotalSupplyQC = 0

	if err := params.Validate(); err == nil {
		t.Fatal("zero total supply must be rejected")
	}
}

func TestParamsValidateRejectsInvalidSubsidy(t *testing.T) {
	params := LegacyParamsV1()
	params.InitialSubsidyQC = 0

	if err := params.Validate(); err == nil {
		t.Fatal("zero subsidy must be rejected")
	}
}

func TestParamsValidateRejectsSubsidyAboveSupply(t *testing.T) {
	params := LegacyParamsV1()
	params.InitialSubsidyQC = params.TotalSupplyQC + 1

	if err := params.Validate(); err == nil {
		t.Fatal("subsidy above total supply must be rejected")
	}
}

func TestParamsValidateRejectsZeroHalvingInterval(t *testing.T) {
	params := LegacyParamsV1()
	params.HalvingIntervalBlocks = 0

	if err := params.Validate(); err == nil {
		t.Fatal("zero halving interval must be rejected")
	}
}

func TestParamsValidateRejectsInvalidBlockTime(t *testing.T) {
	params := LegacyParamsV1()
	params.TargetBlockTime = 0

	if err := params.Validate(); err == nil {
		t.Fatal("zero target block time must be rejected")
	}
}
