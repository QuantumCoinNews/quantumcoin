package config

import "testing"

func TestDefaultTestnetConfigSkeletonV1Valid(t *testing.T) {
	c := DefaultTestnetConfigSkeletonV1()
	if err := ValidateTestnetConfigSkeletonV1(c); err != nil {
		t.Fatalf("default testnet skeleton invalid: %v", err)
	}
}

func TestTestnetConfigSkeletonRejectsMainnetChainFileV1(t *testing.T) {
	c := DefaultTestnetConfigSkeletonV1()
	c.ChainFile = QCMainnetChainFileV1
	if err := ValidateTestnetConfigSkeletonV1(c); err == nil {
		t.Fatalf("expected mainnet chain file rejection")
	}
}

func TestTestnetConfigSkeletonRejectsMainPortsV1(t *testing.T) {
	c := DefaultTestnetConfigSkeletonV1()
	c.HTTPPort = ":8082"
	if err := ValidateTestnetConfigSkeletonV1(c); err == nil {
		t.Fatalf("expected main API port rejection")
	}

	c = DefaultTestnetConfigSkeletonV1()
	c.P2PPort = ":3001"
	if err := ValidateTestnetConfigSkeletonV1(c); err == nil {
		t.Fatalf("expected main P2P port rejection")
	}
}

func TestTestnetConfigSkeletonPreservesSupplyV1(t *testing.T) {
	c := DefaultTestnetConfigSkeletonV1()
	c.TotalSupply = 25_500_001
	if err := ValidateTestnetConfigSkeletonV1(c); err == nil {
		t.Fatalf("expected total supply mismatch rejection")
	}
}

func TestTestnetConfigSkeletonRequiresPublicSafetyV1(t *testing.T) {
	c := DefaultTestnetConfigSkeletonV1()
	c.PublicAPIRequiresToken = false
	if err := ValidateTestnetConfigSkeletonV1(c); err == nil {
		t.Fatalf("expected API token safety rejection")
	}

	c = DefaultTestnetConfigSkeletonV1()
	c.PublicCORSRequiresAllowlist = false
	if err := ValidateTestnetConfigSkeletonV1(c); err == nil {
		t.Fatalf("expected CORS allowlist safety rejection")
	}

	c = DefaultTestnetConfigSkeletonV1()
	c.DevMinerEndpointsDisabled = false
	if err := ValidateTestnetConfigSkeletonV1(c); err == nil {
		t.Fatalf("expected dev/miner endpoint safety rejection")
	}
}

func TestTestnetConfigSkeletonKeepsFaucetPremineDisabledV1(t *testing.T) {
	c := DefaultTestnetConfigSkeletonV1()
	c.FaucetEnabled = true
	if err := ValidateTestnetConfigSkeletonV1(c); err == nil {
		t.Fatalf("expected faucet enabled rejection in skeleton")
	}

	c = DefaultTestnetConfigSkeletonV1()
	c.PremineEnabled = true
	if err := ValidateTestnetConfigSkeletonV1(c); err == nil {
		t.Fatalf("expected premine enabled rejection in skeleton")
	}
}

func TestTestnetEnvSkeletonV1(t *testing.T) {
	env := TestnetEnvSkeletonV1()

	if env["QC_TESTNET"] != "1" {
		t.Fatalf("QC_TESTNET env missing")
	}
	if env["QC_NET_MAGIC"] != QCTestnetMagicV1 {
		t.Fatalf("QC_NET_MAGIC mismatch: %q", env["QC_NET_MAGIC"])
	}
	if env["QC_CHAIN_FILE"] == QCMainnetChainFileV1 {
		t.Fatalf("testnet env must not use mainnet chain file")
	}
	if env["QC_API_TOKEN"] == "" {
		t.Fatalf("token placeholder must exist")
	}
	if env["QC_CORS_ALLOWED_ORIGINS"] == "" {
		t.Fatalf("CORS allowlist placeholder must exist")
	}
}
