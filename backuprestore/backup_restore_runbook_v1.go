// Package backuprestore contains QuantumCoin project code for this module.
package backuprestore

import (
	"errors"
	"strings"
)

const BackupRestoreRunbookVersionV1 = "QC_BACKUP_RESTORE_RUNBOOK_V1"

type ArtifactKind string

const (
	ArtifactChainFile       ArtifactKind = "chain_file"
	ArtifactSQLiteDB        ArtifactKind = "sqlite_db"
	ArtifactJSONLedger      ArtifactKind = "json_ledger"
	ArtifactJSONLAppendOnly ArtifactKind = "jsonl_append_only"
	ArtifactConfig          ArtifactKind = "config"
	ArtifactWalletKey       ArtifactKind = "wallet_key"
)

type BackupArtifact struct {
	Name                  string       `json:"name"`
	Kind                  ArtifactKind `json:"kind"`
	TestnetSeparated      bool         `json:"testnetSeparated"`
	MainnetSeparated      bool         `json:"mainnetSeparated"`
	RequiresHash          bool         `json:"requiresHash"`
	RequiresSizeCheck     bool         `json:"requiresSizeCheck"`
	RequiresManifestEntry bool         `json:"requiresManifestEntry"`
	RequiresEncryption    bool         `json:"requiresEncryption"`
	AppendOnlyPreferred   bool         `json:"appendOnlyPreferred"`
	RestoreDryRunRequired bool         `json:"restoreDryRunRequired"`
	AdminApprovalRequired bool         `json:"adminApprovalRequired"`
	LiveRestoreForbidden  bool         `json:"liveRestoreForbidden"`
}

type Runbook struct {
	Version      string            `json:"version"`
	Mode         string            `json:"mode"`
	Artifacts    []BackupArtifact  `json:"artifacts"`
	SafeBoundary map[string]bool   `json:"safeBoundary"`
	Notes        map[string]string `json:"notes"`
}

func DefaultRunbookV1() Runbook {
	return Runbook{
		Version: BackupRestoreRunbookVersionV1,
		Mode:    "RUNBOOK_ONLY_NO_LIVE_RESTORE",
		Artifacts: []BackupArtifact{
			{
				Name:                  "quantumcoin_chain_file",
				Kind:                  ArtifactChainFile,
				TestnetSeparated:      true,
				MainnetSeparated:      true,
				RequiresHash:          true,
				RequiresSizeCheck:     true,
				RequiresManifestEntry: true,
				RequiresEncryption:    false,
				AppendOnlyPreferred:   true,
				RestoreDryRunRequired: true,
				AdminApprovalRequired: true,
				LiveRestoreForbidden:  true,
			},
			{
				Name:                  "telegram_game_sqlite_db",
				Kind:                  ArtifactSQLiteDB,
				TestnetSeparated:      true,
				MainnetSeparated:      true,
				RequiresHash:          true,
				RequiresSizeCheck:     true,
				RequiresManifestEntry: true,
				RequiresEncryption:    false,
				AppendOnlyPreferred:   false,
				RestoreDryRunRequired: true,
				AdminApprovalRequired: true,
				LiveRestoreForbidden:  true,
			},
			{
				Name:                  "tgwt_ledger",
				Kind:                  ArtifactJSONLedger,
				TestnetSeparated:      true,
				MainnetSeparated:      true,
				RequiresHash:          true,
				RequiresSizeCheck:     true,
				RequiresManifestEntry: true,
				RequiresEncryption:    false,
				AppendOnlyPreferred:   true,
				RestoreDryRunRequired: true,
				AdminApprovalRequired: true,
				LiveRestoreForbidden:  true,
			},
			{
				Name:                  "monthly_nft_review_log",
				Kind:                  ArtifactJSONLedger,
				TestnetSeparated:      true,
				MainnetSeparated:      true,
				RequiresHash:          true,
				RequiresSizeCheck:     true,
				RequiresManifestEntry: true,
				RequiresEncryption:    false,
				AppendOnlyPreferred:   true,
				RestoreDryRunRequired: true,
				AdminApprovalRequired: true,
				LiveRestoreForbidden:  true,
			},
			{
				Name:                  "admin_event_log",
				Kind:                  ArtifactJSONLAppendOnly,
				TestnetSeparated:      true,
				MainnetSeparated:      true,
				RequiresHash:          true,
				RequiresSizeCheck:     true,
				RequiresManifestEntry: true,
				RequiresEncryption:    false,
				AppendOnlyPreferred:   true,
				RestoreDryRunRequired: true,
				AdminApprovalRequired: true,
				LiveRestoreForbidden:  true,
			},
			{
				Name:                  "monitoring_event_log",
				Kind:                  ArtifactJSONLAppendOnly,
				TestnetSeparated:      true,
				MainnetSeparated:      true,
				RequiresHash:          true,
				RequiresSizeCheck:     true,
				RequiresManifestEntry: true,
				RequiresEncryption:    false,
				AppendOnlyPreferred:   true,
				RestoreDryRunRequired: true,
				AdminApprovalRequired: true,
				LiveRestoreForbidden:  true,
			},
			{
				Name:                  "runtime_config_examples",
				Kind:                  ArtifactConfig,
				TestnetSeparated:      true,
				MainnetSeparated:      true,
				RequiresHash:          true,
				RequiresSizeCheck:     true,
				RequiresManifestEntry: true,
				RequiresEncryption:    false,
				AppendOnlyPreferred:   false,
				RestoreDryRunRequired: true,
				AdminApprovalRequired: true,
				LiveRestoreForbidden:  true,
			},
		},
		SafeBoundary: SafeBoundaryV1(),
		Notes: map[string]string{
			"dev":     "Runbook only. No live restore is executed in this step.",
			"testnet": "Testnet backup names, manifests, and restore dry-runs must be separated from mainnet.",
			"mainnet": "Mainnet restore requires explicit admin approval, manifest verification, hash verification, dry-run, and downtime plan.",
			"keys":    "Private keys are never shared in reports. If key backup is needed, use encrypted external custody/manual procedure only.",
		},
	}
}

func SafeBoundaryV1() map[string]bool {
	return map[string]bool{
		"doesNotRunLiveRestore":  true,
		"doesNotDeleteData":      true,
		"doesNotOverwriteData":   true,
		"doesNotMint":            true,
		"doesNotSendFunds":       true,
		"doesNotChangeBalances":  true,
		"doesNotChangeChain":     true,
		"doesNotChangeConsensus": true,
		"doesNotChangeSupply":    true,
		"doesNotStartMining":     true,
		"doesNotStopMining":      true,
		"runbookOnly":            true,
		"requiresManifest":       true,
		"requiresHashCheck":      true,
		"requiresDryRun":         true,
		"requiresAdminApproval":  true,
	}
}

func ValidateRunbookV1(runbook Runbook) error {
	if strings.TrimSpace(runbook.Version) != BackupRestoreRunbookVersionV1 {
		return errors.New("invalid backup restore runbook version")
	}
	if strings.TrimSpace(runbook.Mode) != "RUNBOOK_ONLY_NO_LIVE_RESTORE" {
		return errors.New("invalid backup restore runbook mode")
	}
	if len(runbook.Artifacts) < 1 {
		return errors.New("backup artifacts required")
	}

	seen := map[string]bool{}
	for _, a := range runbook.Artifacts {
		name := strings.TrimSpace(a.Name)
		if name == "" {
			return errors.New("artifact name required")
		}
		if seen[name] {
			return errors.New("duplicate artifact: " + name)
		}
		seen[name] = true

		if strings.TrimSpace(string(a.Kind)) == "" {
			return errors.New("artifact kind required: " + name)
		}
		if !a.TestnetSeparated || !a.MainnetSeparated {
			return errors.New("testnet/mainnet backup separation required: " + name)
		}
		if !a.RequiresHash || !a.RequiresSizeCheck || !a.RequiresManifestEntry {
			return errors.New("hash/size/manifest checks required: " + name)
		}
		if !a.RestoreDryRunRequired {
			return errors.New("restore dry-run required: " + name)
		}
		if !a.AdminApprovalRequired {
			return errors.New("admin approval required: " + name)
		}
		if !a.LiveRestoreForbidden {
			return errors.New("live restore must be forbidden in this runbook step: " + name)
		}
	}
	if err := validateExactSafeBoundaryMapV1(
		runbook.SafeBoundary,
		SafeBoundaryV1(),
	); err != nil {
		return err
	}

	for _, k := range []string{
		"doesNotRunLiveRestore",
		"doesNotDeleteData",
		"doesNotOverwriteData",
		"doesNotMint",
		"doesNotSendFunds",
		"doesNotChangeBalances",
		"doesNotChangeChain",
		"doesNotChangeConsensus",
		"doesNotChangeSupply",
		"doesNotStartMining",
		"doesNotStopMining",
		"runbookOnly",
		"requiresManifest",
		"requiresHashCheck",
		"requiresDryRun",
		"requiresAdminApproval",
	} {
		if !runbook.SafeBoundary[k] {
			return errors.New("safe boundary missing: " + k)
		}
	}

	return nil
}

func RequiredArtifactNamesV1() []string {
	return []string{
		"quantumcoin_chain_file",
		"telegram_game_sqlite_db",
		"tgwt_ledger",
		"monthly_nft_review_log",
		"admin_event_log",
		"monitoring_event_log",
		"runtime_config_examples",
	}
}

func HasRequiredArtifactsV1(runbook Runbook) error {
	seen := map[string]bool{}
	for _, a := range runbook.Artifacts {
		seen[a.Name] = true
	}
	for _, name := range RequiredArtifactNamesV1() {
		if !seen[name] {
			return errors.New("missing required artifact: " + name)
		}
	}
	return nil
}

func validateExactSafeBoundaryMapV1(
	boundary map[string]bool,
	expected map[string]bool,
) error {
	if boundary == nil {
		return errors.New("backup restore runbook safe boundary required")
	}

	if len(boundary) != len(expected) {
		return errors.New("backup restore runbook safe boundary key count mismatch")
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
