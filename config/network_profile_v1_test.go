package config

import "testing"

func clearNetworkProfileEnvV1(t *testing.T) {
	t.Helper()

	keys := []string{
		"QC_TESTNET",
		"QC_TESTNET_PUBLIC_API",
		"QC_API_PUBLIC_TESTNET",
		"QC_PUBLIC_TESTNET",
		"QC_MAINNET",
		"QC_NET_MAGIC",
		"QC_NETWORK",
		"QC_NETWORK_MODE",
		"QC_CHAIN_MODE",
		"QC_MODE",
		"QC_DIFFICULTY_BITS",
	}

	for _, key := range keys {
		t.Setenv(key, "")
	}
}

func TestResolveNetworkProfileV1RejectsMalformedAndConflictingAliases(t *testing.T) {
	for _, scenario := range []struct {
		name string
		env  map[string]string
	}{
		{"typo", map[string]string{"QC_NETWORK": "not-mainnet"}},
		{"boolean", map[string]string{"QC_TESTNET": "enabled"}},
		{"conflict", map[string]string{"QC_NETWORK": "local-release", "QC_MODE": "testnet"}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			clearNetworkProfileEnvV1(t)
			for key, value := range scenario.env {
				t.Setenv(key, value)
			}
			if _, err := ResolveNetworkProfileV1(QCMainnetChainFileV1); err == nil {
				t.Fatal("invalid network settings accepted")
			}
		})
	}
}

func TestResolveNetworkProfileV1DefaultsToLocalRelease(
	t *testing.T,
) {
	clearNetworkProfileEnvV1(t)

	info, err := ResolveNetworkProfileV1(
		QCMainnetChainFileV1,
	)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if info.Profile != QCNetworkProfileLocalReleaseV1 {
		t.Fatalf("unexpected profile: %s", info.Profile)
	}

	if !info.AllowsFixedDifficultyOverride {
		t.Fatal("local release must preserve current fixed override compatibility")
	}

	if info.CanonicalDefaultDifficultyBits != 16 {
		t.Fatalf(
			"local release base difficulty: %d",
			info.CanonicalDefaultDifficultyBits,
		)
	}
}

func TestResolveNetworkProfileV1ExplicitMainnet(
	t *testing.T,
) {
	clearNetworkProfileEnvV1(t)
	t.Setenv("QC_NETWORK", "mainnet")

	info, err := ResolveNetworkProfileV1(
		QCMainnetChainFileV1,
	)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if info.Profile != QCNetworkProfileMainnetV1 {
		t.Fatalf("unexpected profile: %s", info.Profile)
	}

	if !info.IsExplicitMainnet {
		t.Fatal("mainnet must be explicit")
	}

	if info.AllowsFixedDifficultyOverride {
		t.Fatal("mainnet must not allow fixed difficulty override")
	}
}

func TestResolveNetworkProfileV1LocalTestnet(
	t *testing.T,
) {
	clearNetworkProfileEnvV1(t)

	t.Setenv("QC_TESTNET", "1")
	t.Setenv("QC_PUBLIC_TESTNET", "0")

	info, err := ResolveNetworkProfileV1(
		QCTestnetChainFileV1,
	)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if info.Profile != QCNetworkProfileTestnetLocalV1 {
		t.Fatalf("unexpected profile: %s", info.Profile)
	}

	if info.CanonicalDefaultDifficultyBits != 8 {
		t.Fatalf(
			"testnet base difficulty: %d",
			info.CanonicalDefaultDifficultyBits,
		)
	}

	if info.AllowsFixedDifficultyOverride {
		t.Fatal("testnet must not allow inherited fixed override")
	}
}

func TestResolveNetworkProfileV1PublicTestnet(
	t *testing.T,
) {
	clearNetworkProfileEnvV1(t)

	t.Setenv("QC_TESTNET", "1")
	t.Setenv("QC_PUBLIC_TESTNET", "1")

	info, err := ResolveNetworkProfileV1(
		QCTestnetChainFileV1,
	)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if info.Profile != QCNetworkProfileTestnetPublicV1 {
		t.Fatalf("unexpected profile: %s", info.Profile)
	}

	if !info.IsPublicTestnet {
		t.Fatal("public testnet flag lost")
	}
}

func TestResolveNetworkProfileV1DetectsTestnetChainName(
	t *testing.T,
) {
	clearNetworkProfileEnvV1(t)

	info, err := ResolveNetworkProfileV1(
		"data/testnet/chain_data_testnet.dat",
	)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if info.Profile != QCNetworkProfileTestnetLocalV1 {
		t.Fatalf("unexpected profile: %s", info.Profile)
	}
}

func TestResolveNetworkProfileV1RejectsConflictingSignals(
	t *testing.T,
) {
	clearNetworkProfileEnvV1(t)

	t.Setenv("QC_TESTNET", "1")
	t.Setenv("QC_NETWORK", "mainnet")

	_, err := ResolveNetworkProfileV1(
		QCTestnetChainFileV1,
	)
	if err == nil {
		t.Fatal("expected conflicting network signal rejection")
	}
}

func TestResolveNetworkProfileV1DifficultyOverrideDoesNotSelectNetwork(
	t *testing.T,
) {
	clearNetworkProfileEnvV1(t)

	t.Setenv("QC_DIFFICULTY_BITS", "12")

	info, err := ResolveNetworkProfileV1(
		QCMainnetChainFileV1,
	)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if info.Profile != QCNetworkProfileLocalReleaseV1 {
		t.Fatalf(
			"difficulty override selected network profile: %s",
			info.Profile,
		)
	}
}
