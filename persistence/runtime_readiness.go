package persistence

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DomainRuntimeEvidenceV1 contains concrete runtime artifacts for one
// persistence domain. Boolean policy declarations are not accepted as proof.
type DomainRuntimeEvidenceV1 struct {
	DomainName          string
	DomainPath          string
	DataPath            string
	ObservedSchema      string
	BackupManifest      BackupManifestV1
	RestoreTarget       string
	MigrationFromSchema string
}

// RuntimeValidationHooksV1 binds storage-specific verification to the common
// readiness validator. A required nil hook fails closed.
type RuntimeValidationHooksV1 struct {
	VerifySchema func(
		domain DataDomain,
		dataPath string,
		observedSchema string,
	) error

	VerifyEncryption func(
		domain DataDomain,
		dataPath string,
	) error

	VerifyAppendOnly func(
		domain DataDomain,
		dataPath string,
	) error

	VerifyMigration func(
		domain DataDomain,
		dataPath string,
		fromSchema string,
		toSchema string,
	) error
}

// ValidateRuntimeReadinessV1 verifies concrete persistence artifacts for every
// domain in plan. It does not infer readiness from plan metadata.
func ValidateRuntimeReadinessV1(
	plan Plan,
	cfg StorageConfig,
	evidence []DomainRuntimeEvidenceV1,
	hooks RuntimeValidationHooksV1,
) error {
	if err := ValidatePlanV1(plan); err != nil {
		return fmt.Errorf("validate persistence plan: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate storage config: %w", err)
	}

	if len(evidence) != len(plan.Domains) {
		return fmt.Errorf(
			"runtime evidence count mismatch: got=%d want=%d",
			len(evidence),
			len(plan.Domains),
		)
	}

	if hooks.VerifySchema == nil {
		return errors.New("runtime schema verifier required")
	}

	if hooks.VerifyEncryption == nil {
		return errors.New("runtime encryption verifier required")
	}

	evidenceByDomain := make(
		map[string]DomainRuntimeEvidenceV1,
		len(evidence),
	)

	for _, item := range evidence {
		name := strings.TrimSpace(item.DomainName)
		if name == "" {
			return errors.New(
				"runtime evidence domain name required",
			)
		}

		if name != item.DomainName {
			return fmt.Errorf(
				"runtime evidence domain name contains "+
					"surrounding whitespace: %q",
				item.DomainName,
			)
		}

		if _, exists := evidenceByDomain[name]; exists {
			return fmt.Errorf(
				"duplicate runtime evidence for domain: %s",
				name,
			)
		}

		evidenceByDomain[name] = item
	}

	for _, domain := range plan.Domains {
		item, exists := evidenceByDomain[domain.Name]
		if !exists {
			return fmt.Errorf(
				"runtime evidence missing for domain: %s",
				domain.Name,
			)
		}

		if err := validateDomainRuntimeEvidenceV1(
			cfg,
			domain,
			item,
			hooks,
		); err != nil {
			return fmt.Errorf(
				"runtime readiness failed for domain %s: %w",
				domain.Name,
				err,
			)
		}
	}

	return nil
}

func validateDomainRuntimeEvidenceV1(
	cfg StorageConfig,
	domain DataDomain,
	evidence DomainRuntimeEvidenceV1,
	hooks RuntimeValidationHooksV1,
) error {
	expectedDomainPath, err := ResolveDomainPath(cfg, domain)
	if err != nil {
		return fmt.Errorf("resolve domain path: %w", err)
	}

	domainPath, err := cleanAbsolutePathV1(
		evidence.DomainPath,
		"domain path",
	)
	if err != nil {
		return err
	}

	expectedDomainPath, err = filepath.Abs(expectedDomainPath)
	if err != nil {
		return fmt.Errorf(
			"resolve absolute expected domain path: %w",
			err,
		)
	}
	expectedDomainPath = filepath.Clean(expectedDomainPath)

	if !samePathV1(domainPath, expectedDomainPath) {
		return fmt.Errorf(
			"domain path mismatch: got=%s want=%s",
			domainPath,
			expectedDomainPath,
		)
	}

	domainInfo, err := os.Stat(domainPath)
	if err != nil {
		return fmt.Errorf("inspect domain path: %w", err)
	}

	if !domainInfo.IsDir() {
		return errors.New("domain path is not a directory")
	}

	dataPath, err := cleanAbsolutePathV1(
		evidence.DataPath,
		"data path",
	)
	if err != nil {
		return err
	}

	if !pathWithinV1(domainPath, dataPath) {
		return fmt.Errorf(
			"data path escapes domain path: %s",
			dataPath,
		)
	}

	dataInfo, err := os.Stat(dataPath)
	if err != nil {
		return fmt.Errorf("inspect persistence data: %w", err)
	}

	if !dataInfo.Mode().IsRegular() {
		return errors.New(
			"persistence data path is not a regular file",
		)
	}

	observedSchema := strings.TrimSpace(
		evidence.ObservedSchema,
	)
	if observedSchema == "" {
		return errors.New("observed schema version required")
	}

	if observedSchema != evidence.ObservedSchema {
		return errors.New(
			"observed schema contains surrounding whitespace",
		)
	}

	if observedSchema != domain.SchemaVersion {
		return fmt.Errorf(
			"schema version mismatch: got=%s want=%s",
			observedSchema,
			domain.SchemaVersion,
		)
	}

	if err := hooks.VerifySchema(
		domain,
		dataPath,
		observedSchema,
	); err != nil {
		return fmt.Errorf("verify schema: %w", err)
	}

	if err := hooks.VerifyEncryption(
		domain,
		dataPath,
	); err != nil {
		return fmt.Errorf("verify encryption: %w", err)
	}

	if domain.AppendOnlyLedger {
		if hooks.VerifyAppendOnly == nil {
			return errors.New(
				"append-only verifier required",
			)
		}

		if err := hooks.VerifyAppendOnly(
			domain,
			dataPath,
		); err != nil {
			return fmt.Errorf(
				"verify append-only behavior: %w",
				err,
			)
		}
	}

	if err := validateBackupAndRestoreV1(
		dataPath,
		evidence,
	); err != nil {
		return err
	}

	if domain.MigrationRequired {
		if hooks.VerifyMigration == nil {
			return errors.New(
				"migration verifier required",
			)
		}

		fromSchema := strings.TrimSpace(
			evidence.MigrationFromSchema,
		)
		if fromSchema == "" {
			return errors.New(
				"migration source schema required",
			)
		}

		if fromSchema != evidence.MigrationFromSchema {
			return errors.New(
				"migration source schema contains " +
					"surrounding whitespace",
			)
		}

		if fromSchema == domain.SchemaVersion {
			return errors.New(
				"migration source and target schemas are equal",
			)
		}

		if err := hooks.VerifyMigration(
			domain,
			dataPath,
			fromSchema,
			domain.SchemaVersion,
		); err != nil {
			return fmt.Errorf(
				"verify migration compatibility: %w",
				err,
			)
		}
	}

	return nil
}

func validateBackupAndRestoreV1(
	dataPath string,
	evidence DomainRuntimeEvidenceV1,
) error {
	manifest := evidence.BackupManifest

	if manifest.Version != BackupManifestVersionV1 {
		return fmt.Errorf(
			"invalid backup manifest version: %q",
			manifest.Version,
		)
	}

	sourcePath, err := cleanAbsolutePathV1(
		manifest.Source,
		"backup source path",
	)
	if err != nil {
		return err
	}

	if !samePathV1(sourcePath, dataPath) {
		return fmt.Errorf(
			"backup source mismatch: got=%s want=%s",
			sourcePath,
			dataPath,
		)
	}

	backupPath, err := cleanAbsolutePathV1(
		manifest.Backup,
		"backup path",
	)
	if err != nil {
		return err
	}

	if samePathV1(backupPath, dataPath) {
		return errors.New(
			"backup path must differ from data path",
		)
	}

	if manifest.Size < 0 {
		return errors.New("invalid backup size")
	}

	if strings.TrimSpace(manifest.SHA256) == "" {
		return errors.New("backup SHA256 required")
	}

	if err := VerifyBackupFileV1(
		backupPath,
		manifest.SHA256,
		manifest.Size,
	); err != nil {
		return fmt.Errorf("verify backup artifact: %w", err)
	}

	restoreTarget, err := cleanAbsolutePathV1(
		evidence.RestoreTarget,
		"restore target path",
	)
	if err != nil {
		return err
	}

	if samePathV1(restoreTarget, dataPath) {
		return errors.New(
			"restore target must differ from data path",
		)
	}

	if samePathV1(restoreTarget, backupPath) {
		return errors.New(
			"restore target must differ from backup path",
		)
	}

	dryRun, err := RestoreFileDryRunV1(
		backupPath,
		manifest.SHA256,
		manifest.Size,
		restoreTarget,
	)
	if err != nil {
		return fmt.Errorf("restore dry-run: %w", err)
	}

	if dryRun.Version != BackupManifestVersionV1 ||
		dryRun.Size != manifest.Size ||
		dryRun.SHA256 != manifest.SHA256 {
		return errors.New(
			"restore dry-run result does not match backup manifest",
		)
	}

	if !samePathV1(dryRun.Source, backupPath) {
		return errors.New(
			"restore dry-run source mismatch",
		)
	}

	if !samePathV1(dryRun.Backup, restoreTarget) {
		return errors.New(
			"restore dry-run target mismatch",
		)
	}

	return nil
}

func cleanAbsolutePathV1(
	path string,
	label string,
) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("%s required", label)
	}

	if path != strings.TrimSpace(path) {
		return "", fmt.Errorf(
			"%s contains surrounding whitespace",
			label,
		)
	}

	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("%s must be absolute", label)
	}

	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf(
			"resolve absolute %s: %w",
			label,
			err,
		)
	}

	return filepath.Clean(absolute), nil
}

func pathWithinV1(parent string, child string) bool {
	relative, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}

	if relative == "." {
		return false
	}

	return relative != ".." &&
		!strings.HasPrefix(
			relative,
			".."+string(filepath.Separator),
		)
}

func samePathV1(left string, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)

	return strings.EqualFold(left, right)
}
