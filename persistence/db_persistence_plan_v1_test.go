package persistence

import "testing"

func TestDefaultDBPersistencePlanValid(t *testing.T) {
	plan := DefaultPlanV1()

	if err := ValidatePlanV1(plan); err != nil {
		t.Fatalf("default plan should validate: %v", err)
	}

	if err := HasRequiredDomainsV1(plan); err != nil {
		t.Fatalf(
			"default plan missing required domain: %v",
			err,
		)
	}
}

func TestDBPersistenceSafeBoundary(t *testing.T) {
	plan := DefaultPlanV1()

	for _, key := range []string{
		"doesNotMint",
		"doesNotSendFunds",
		"doesNotChangeBalances",
		"doesNotChangeChain",
		"doesNotChangeConsensus",
		"doesNotChangeSupply",
		"doesNotStartMining",
		"doesNotStopMining",
		"planOnly",
		"requiresBackupPlan",
		"requiresMigrationPlan",
		"requiresRestoreTest",
	} {
		if !plan.SafeBoundary[key] {
			t.Fatalf(
				"safe boundary missing: %s",
				key,
			)
		}
	}
}

func TestDBPersistenceRejectsMissingSeparation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*DataDomain)
	}{
		{
			name: "testnet",
			mutate: func(domain *DataDomain) {
				domain.TestnetSeparated = false
			},
		},
		{
			name: "mainnet",
			mutate: func(domain *DataDomain) {
				domain.MainnetSeparated = false
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := DefaultPlanV1()
			test.mutate(&plan.Domains[0])

			if err := ValidatePlanV1(plan); err == nil {
				t.Fatal(
					"missing network separation must fail",
				)
			}
		})
	}
}

func TestDBPersistenceRejectsDuplicateDomain(t *testing.T) {
	plan := DefaultPlanV1()
	plan.Domains = append(
		plan.Domains,
		plan.Domains[0],
	)

	if err := ValidatePlanV1(plan); err == nil {
		t.Fatal("duplicate domain must fail")
	}
}

func TestDBPersistenceRejectsUnknownStorageMode(t *testing.T) {
	plan := DefaultPlanV1()
	plan.Domains[0].StorageMode = StorageMode(
		"unknown_storage",
	)

	if err := ValidatePlanV1(plan); err == nil {
		t.Fatal("unknown storage mode must fail")
	}
}

func TestSupportedStorageModesV1(t *testing.T) {
	supported := []StorageMode{
		StorageModeSQLite,
		StorageModePostgres,
		StorageModeFile,
		StorageModeAppendOnly,
	}

	for _, mode := range supported {
		if !IsSupportedStorageModeV1(mode) {
			t.Fatalf(
				"expected supported storage mode: %q",
				mode,
			)
		}
	}

	for _, mode := range []StorageMode{
		"",
		"memory",
		"mysql",
		"SQLite",
		" sqlite ",
	} {
		if IsSupportedStorageModeV1(mode) {
			t.Fatalf(
				"unexpected supported storage mode: %q",
				mode,
			)
		}
	}
}

func TestDBPersistenceRejectsInvalidSchemaVersion(t *testing.T) {
	invalidVersions := []string{
		"",
		" ",
		"v1",
		"chain",
		"chain_v0",
		"chain_v",
		"Chain_V1",
		"chain-v1",
		"chain_v1 ",
		" chain_v1",
	}

	for _, schemaVersion := range invalidVersions {
		t.Run(schemaVersion, func(t *testing.T) {
			plan := DefaultPlanV1()
			plan.Domains[0].SchemaVersion = schemaVersion

			if err := ValidatePlanV1(plan); err == nil {
				t.Fatalf(
					"invalid schema version must fail: %q",
					schemaVersion,
				)
			}
		})
	}
}

func TestDBPersistenceRejectsAppendOnlyModeWithoutLedger(
	t *testing.T,
) {
	plan := DefaultPlanV1()
	plan.Domains[0].StorageMode = StorageModeAppendOnly
	plan.Domains[0].AppendOnlyLedger = false

	if err := ValidatePlanV1(plan); err == nil {
		t.Fatal(
			"append-only mode without append-only ledger must fail",
		)
	}
}

func TestDBPersistenceRejectsMissingBackupRequirement(
	t *testing.T,
) {
	plan := DefaultPlanV1()
	plan.Domains[0].BackupRequired = false

	if err := ValidatePlanV1(plan); err == nil {
		t.Fatal("domain without backup requirement must fail")
	}
}

func TestDBPersistenceRejectsWhitespaceDomainName(
	t *testing.T,
) {
	plan := DefaultPlanV1()
	plan.Domains[0].Name = " quantumcoin_chain "

	if err := ValidatePlanV1(plan); err == nil {
		t.Fatal("domain name with surrounding whitespace must fail")
	}
}

func TestDBPersistenceRejectsMissingRequiredDomain(
	t *testing.T,
) {
	plan := DefaultPlanV1()
	plan.Domains = plan.Domains[1:]

	if err := ValidatePlanV1(plan); err == nil {
		t.Fatal("missing required domain must fail validation")
	}

	if err := HasRequiredDomainsV1(plan); err == nil {
		t.Fatal("required-domain check must fail")
	}
}

func TestDBPersistenceRejectsIncompleteSafeBoundary(
	t *testing.T,
) {
	requiredKeys := []string{
		"doesNotMint",
		"doesNotSendFunds",
		"doesNotChangeBalances",
		"doesNotChangeChain",
		"doesNotChangeConsensus",
		"doesNotChangeSupply",
		"doesNotStartMining",
		"doesNotStopMining",
		"planOnly",
		"requiresBackupPlan",
		"requiresMigrationPlan",
		"requiresRestoreTest",
	}

	for _, key := range requiredKeys {
		t.Run(key, func(t *testing.T) {
			plan := DefaultPlanV1()
			plan.SafeBoundary[key] = false

			if err := ValidatePlanV1(plan); err == nil {
				t.Fatalf(
					"disabled safe boundary must fail: %s",
					key,
				)
			}
		})
	}
}

func TestDBPersistenceRejectsNilSafeBoundary(t *testing.T) {
	plan := DefaultPlanV1()
	plan.SafeBoundary = nil

	if err := ValidatePlanV1(plan); err == nil {
		t.Fatal("nil safe boundary must fail")
	}
}
