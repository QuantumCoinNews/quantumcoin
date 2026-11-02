// Package persistence contains QuantumCoin project code for this module.
package persistence

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const DBPersistencePlanVersionV1 = "QC_DB_PERSISTENCE_PLAN_V1"

type StorageMode string

const (
	StorageModeSQLite     StorageMode = "sqlite"
	StorageModePostgres   StorageMode = "postgres"
	StorageModeFile       StorageMode = "file"
	StorageModeAppendOnly StorageMode = "append_only"
)

var schemaVersionPatternV1 = regexp.MustCompile(
	`^[a-z][a-z0-9_]*_v[1-9][0-9]*$`,
)

type DataDomain struct {
	Name              string      `json:"name"`
	StorageMode       StorageMode `json:"storageMode"`
	SchemaVersion     string      `json:"schemaVersion"`
	TestnetSeparated  bool        `json:"testnetSeparated"`
	MainnetSeparated  bool        `json:"mainnetSeparated"`
	BackupRequired    bool        `json:"backupRequired"`
	MigrationRequired bool        `json:"migrationRequired"`
	AppendOnlyLedger  bool        `json:"appendOnlyLedger"`
}

type Plan struct {
	Version      string            `json:"version"`
	Domains      []DataDomain      `json:"domains"`
	SafeBoundary map[string]bool   `json:"safeBoundary"`
	Notes        map[string]string `json:"notes"`
}

func DefaultPlanV1() Plan {
	return Plan{
		Version: DBPersistencePlanVersionV1,
		Domains: []DataDomain{
			{
				Name:              "quantumcoin_chain",
				StorageMode:       StorageModeFile,
				SchemaVersion:     "chain_file_v1",
				TestnetSeparated:  true,
				MainnetSeparated:  true,
				BackupRequired:    true,
				MigrationRequired: false,
				AppendOnlyLedger:  true,
			},
			{
				Name:              "telegram_game_state",
				StorageMode:       StorageModeSQLite,
				SchemaVersion:     "tg_game_db_v1",
				TestnetSeparated:  true,
				MainnetSeparated:  true,
				BackupRequired:    true,
				MigrationRequired: true,
				AppendOnlyLedger:  false,
			},
			{
				Name:              "tgwt_ledger",
				StorageMode:       StorageModeSQLite,
				SchemaVersion:     "tgwt_ledger_v1",
				TestnetSeparated:  true,
				MainnetSeparated:  true,
				BackupRequired:    true,
				MigrationRequired: true,
				AppendOnlyLedger:  true,
			},
			{
				Name:              "withdraw_claim_review",
				StorageMode:       StorageModeSQLite,
				SchemaVersion:     "withdraw_claim_review_v1",
				TestnetSeparated:  true,
				MainnetSeparated:  true,
				BackupRequired:    true,
				MigrationRequired: true,
				AppendOnlyLedger:  true,
			},
			{
				Name:              "monthly_nft_review",
				StorageMode:       StorageModeSQLite,
				SchemaVersion:     "monthly_nft_review_v1",
				TestnetSeparated:  true,
				MainnetSeparated:  true,
				BackupRequired:    true,
				MigrationRequired: true,
				AppendOnlyLedger:  true,
			},
			{
				Name:              "admin_event_log",
				StorageMode:       StorageModeAppendOnly,
				SchemaVersion:     "admin_event_log_v1",
				TestnetSeparated:  true,
				MainnetSeparated:  true,
				BackupRequired:    true,
				MigrationRequired: true,
				AppendOnlyLedger:  true,
			},
			{
				Name:              "monitoring_events",
				StorageMode:       StorageModeAppendOnly,
				SchemaVersion:     "monitoring_events_v1",
				TestnetSeparated:  true,
				MainnetSeparated:  true,
				BackupRequired:    true,
				MigrationRequired: true,
				AppendOnlyLedger:  true,
			},
		},
		SafeBoundary: SafeBoundaryV1(),
		Notes: map[string]string{
			"dev": "Local SQLite/file storage is allowed for " +
				"development and testnet preparation.",
			"testnet": "Testnet must use separated DB names/paths " +
				"and never share mainnet files.",
			"mainnet": "Mainnet requires final migration, backup, " +
				"restore, encryption, and audit-log checks before launch.",
		},
	}
}

func SafeBoundaryV1() map[string]bool {
	return map[string]bool{
		"doesNotMint":            true,
		"doesNotSendFunds":       true,
		"doesNotChangeBalances":  true,
		"doesNotChangeChain":     true,
		"doesNotChangeConsensus": true,
		"doesNotChangeSupply":    true,
		"doesNotStartMining":     true,
		"doesNotStopMining":      true,
		"planOnly":               true,
		"requiresBackupPlan":     true,
		"requiresMigrationPlan":  true,
		"requiresRestoreTest":    true,
	}
}

// IsSupportedStorageModeV1 reports whether mode is part of the closed V1
// storage-mode set. Unknown strings must fail instead of silently becoming a
// new persistence policy.
func IsSupportedStorageModeV1(mode StorageMode) bool {
	switch mode {
	case StorageModeSQLite,
		StorageModePostgres,
		StorageModeFile,
		StorageModeAppendOnly:
		return true
	default:
		return false
	}
}

func ValidatePlanV1(plan Plan) error {
	if strings.TrimSpace(plan.Version) != DBPersistencePlanVersionV1 {
		return errors.New("invalid db persistence plan version")
	}

	if len(plan.Domains) == 0 {
		return errors.New("db persistence domains required")
	}

	seen := make(map[string]struct{}, len(plan.Domains))

	for index, domain := range plan.Domains {
		if err := validateDataDomainV1(domain, index); err != nil {
			return err
		}

		name := strings.TrimSpace(domain.Name)
		if _, exists := seen[name]; exists {
			return fmt.Errorf(
				"duplicate persistence domain: %s",
				name,
			)
		}

		seen[name] = struct{}{}
	}

	if err := validateSafeBoundaryV1(plan.SafeBoundary); err != nil {
		return err
	}

	if err := HasRequiredDomainsV1(plan); err != nil {
		return err
	}

	return nil
}

func validateDataDomainV1(domain DataDomain, index int) error {
	name := strings.TrimSpace(domain.Name)
	if name == "" {
		return fmt.Errorf(
			"domain name required at index %d",
			index,
		)
	}

	if name != domain.Name {
		return fmt.Errorf(
			"domain name must not contain surrounding whitespace: %s",
			name,
		)
	}

	if !IsSupportedStorageModeV1(domain.StorageMode) {
		return fmt.Errorf(
			"unsupported storage mode %q for domain: %s",
			domain.StorageMode,
			name,
		)
	}

	schemaVersion := strings.TrimSpace(domain.SchemaVersion)
	if schemaVersion == "" {
		return fmt.Errorf(
			"schema version required for domain: %s",
			name,
		)
	}

	if schemaVersion != domain.SchemaVersion {
		return fmt.Errorf(
			"schema version must not contain surrounding whitespace "+
				"for domain: %s",
			name,
		)
	}

	if !schemaVersionPatternV1.MatchString(schemaVersion) {
		return fmt.Errorf(
			"invalid schema version %q for domain: %s",
			schemaVersion,
			name,
		)
	}

	if !domain.TestnetSeparated || !domain.MainnetSeparated {
		return fmt.Errorf(
			"testnet/mainnet separation required for domain: %s",
			name,
		)
	}

	if !domain.BackupRequired {
		return fmt.Errorf(
			"backup required for domain: %s",
			name,
		)
	}

	if domain.StorageMode == StorageModeAppendOnly &&
		!domain.AppendOnlyLedger {
		return fmt.Errorf(
			"append-only storage mode requires append-only ledger "+
				"for domain: %s",
			name,
		)
	}

	return nil
}

func validateSafeBoundaryV1(boundary map[string]bool) error {
	if boundary == nil {
		return errors.New("safe boundary required")
	}
	if err := validateExactSafeBoundaryMapV1(
		boundary,
		SafeBoundaryV1(),
	); err != nil {
		return err
	}

	required := []string{
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

	for _, key := range required {
		if !boundary[key] {
			return errors.New("safe boundary missing: " + key)
		}
	}

	return nil
}

func RequiredDomainNamesV1() []string {
	return []string{
		"quantumcoin_chain",
		"telegram_game_state",
		"tgwt_ledger",
		"withdraw_claim_review",
		"monthly_nft_review",
		"admin_event_log",
		"monitoring_events",
	}
}

func HasRequiredDomainsV1(plan Plan) error {
	seen := make(map[string]struct{}, len(plan.Domains))

	for _, domain := range plan.Domains {
		name := strings.TrimSpace(domain.Name)
		if name == "" {
			continue
		}

		seen[name] = struct{}{}
	}

	for _, name := range RequiredDomainNamesV1() {
		if _, exists := seen[name]; !exists {
			return errors.New(
				"missing required persistence domain: " + name,
			)
		}
	}

	return nil
}

func validateExactSafeBoundaryMapV1(
	boundary map[string]bool,
	expected map[string]bool,
) error {
	if boundary == nil {
		return errors.New("persistence plan safe boundary required")
	}

	if len(boundary) != len(expected) {
		return errors.New("persistence plan safe boundary key count mismatch")
	}

	for key, want := range expected {
		got, exists := boundary[key]
		if !exists {
			return errors.New("safe boundary missing: " + key)
		}
		if got != want {
			return errors.New("safe boundary value mismatch: " + key)
		}
	}

	for key := range boundary {
		if _, exists := expected[key]; !exists {
			return errors.New("unexpected safe boundary key: " + key)
		}
	}

	return nil
}
