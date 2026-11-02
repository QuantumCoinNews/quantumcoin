package persistence

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRuntimeReadinessAcceptsVerifiedArtifacts(
	t *testing.T,
) {
	plan, cfg, evidence, hooks := runtimeReadinessFixtureV1(t)

	if err := ValidateRuntimeReadinessV1(
		plan,
		cfg,
		evidence,
		hooks,
	); err != nil {
		t.Fatalf(
			"verified runtime artifacts should pass: %v",
			err,
		)
	}
}

func TestRuntimeReadinessRejectsMissingEvidence(
	t *testing.T,
) {
	plan, cfg, evidence, hooks := runtimeReadinessFixtureV1(t)
	evidence = evidence[:len(evidence)-1]

	if err := ValidateRuntimeReadinessV1(
		plan,
		cfg,
		evidence,
		hooks,
	); err == nil {
		t.Fatal("missing domain evidence must fail")
	}
}

func TestRuntimeReadinessRejectsDuplicateEvidence(
	t *testing.T,
) {
	plan, cfg, evidence, hooks := runtimeReadinessFixtureV1(t)
	evidence[len(evidence)-1] = evidence[0]

	if err := ValidateRuntimeReadinessV1(
		plan,
		cfg,
		evidence,
		hooks,
	); err == nil {
		t.Fatal("duplicate domain evidence must fail")
	}
}

func TestRuntimeReadinessRejectsSchemaMismatch(
	t *testing.T,
) {
	plan, cfg, evidence, hooks := runtimeReadinessFixtureV1(t)
	evidence[0].ObservedSchema = "wrong_schema_v1"

	if err := ValidateRuntimeReadinessV1(
		plan,
		cfg,
		evidence,
		hooks,
	); err == nil {
		t.Fatal("schema mismatch must fail")
	}
}

func TestRuntimeReadinessRejectsDomainPathMismatch(
	t *testing.T,
) {
	plan, cfg, evidence, hooks := runtimeReadinessFixtureV1(t)
	evidence[0].DomainPath = filepath.Join(
		cfg.RootDir,
		"wrong-network",
		plan.Domains[0].Name,
		plan.Domains[0].SchemaVersion,
	)

	if err := ValidateRuntimeReadinessV1(
		plan,
		cfg,
		evidence,
		hooks,
	); err == nil {
		t.Fatal("domain path mismatch must fail")
	}
}

func TestRuntimeReadinessRejectsDataPathEscape(
	t *testing.T,
) {
	plan, cfg, evidence, hooks := runtimeReadinessFixtureV1(t)

	outsidePath := filepath.Join(
		t.TempDir(),
		"outside.gob",
	)
	if err := os.WriteFile(
		outsidePath,
		[]byte("outside"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	evidence[0].DataPath = outsidePath

	if err := ValidateRuntimeReadinessV1(
		plan,
		cfg,
		evidence,
		hooks,
	); err == nil {
		t.Fatal("data path outside domain must fail")
	}
}

func TestRuntimeReadinessRejectsCorruptBackup(
	t *testing.T,
) {
	plan, cfg, evidence, hooks := runtimeReadinessFixtureV1(t)

	if err := os.WriteFile(
		evidence[0].BackupManifest.Backup,
		[]byte("corrupt-backup"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	if err := ValidateRuntimeReadinessV1(
		plan,
		cfg,
		evidence,
		hooks,
	); err == nil {
		t.Fatal("corrupt backup must fail")
	}
}

func TestRuntimeReadinessRejectsExistingRestoreTarget(
	t *testing.T,
) {
	plan, cfg, evidence, hooks := runtimeReadinessFixtureV1(t)

	if err := os.MkdirAll(
		filepath.Dir(evidence[0].RestoreTarget),
		0o700,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		evidence[0].RestoreTarget,
		[]byte("existing-target"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	if err := ValidateRuntimeReadinessV1(
		plan,
		cfg,
		evidence,
		hooks,
	); err == nil {
		t.Fatal("existing restore target must fail")
	}
}

func TestRuntimeReadinessRequiresSchemaVerifier(
	t *testing.T,
) {
	plan, cfg, evidence, hooks := runtimeReadinessFixtureV1(t)
	hooks.VerifySchema = nil

	if err := ValidateRuntimeReadinessV1(
		plan,
		cfg,
		evidence,
		hooks,
	); err == nil {
		t.Fatal("missing schema verifier must fail")
	}
}

func TestRuntimeReadinessRequiresEncryptionVerifier(
	t *testing.T,
) {
	plan, cfg, evidence, hooks := runtimeReadinessFixtureV1(t)
	hooks.VerifyEncryption = nil

	if err := ValidateRuntimeReadinessV1(
		plan,
		cfg,
		evidence,
		hooks,
	); err == nil {
		t.Fatal("missing encryption verifier must fail")
	}
}

func TestRuntimeReadinessRequiresAppendOnlyVerifier(
	t *testing.T,
) {
	plan, cfg, evidence, hooks := runtimeReadinessFixtureV1(t)
	hooks.VerifyAppendOnly = nil

	if err := ValidateRuntimeReadinessV1(
		plan,
		cfg,
		evidence,
		hooks,
	); err == nil {
		t.Fatal("missing append-only verifier must fail")
	}
}

func TestRuntimeReadinessRequiresMigrationVerifier(
	t *testing.T,
) {
	plan, cfg, evidence, hooks := runtimeReadinessFixtureV1(t)
	hooks.VerifyMigration = nil

	if err := ValidateRuntimeReadinessV1(
		plan,
		cfg,
		evidence,
		hooks,
	); err == nil {
		t.Fatal("missing migration verifier must fail")
	}
}

func TestRuntimeReadinessPropagatesVerifierFailure(
	t *testing.T,
) {
	tests := []struct {
		name   string
		mutate func(*RuntimeValidationHooksV1)
	}{
		{
			name: "schema",
			mutate: func(hooks *RuntimeValidationHooksV1) {
				hooks.VerifySchema = func(
					DataDomain,
					string,
					string,
				) error {
					return errors.New("schema verification failed")
				}
			},
		},
		{
			name: "encryption",
			mutate: func(hooks *RuntimeValidationHooksV1) {
				hooks.VerifyEncryption = func(
					DataDomain,
					string,
				) error {
					return errors.New(
						"encryption verification failed",
					)
				}
			},
		},
		{
			name: "append-only",
			mutate: func(hooks *RuntimeValidationHooksV1) {
				hooks.VerifyAppendOnly = func(
					DataDomain,
					string,
				) error {
					return errors.New(
						"append-only verification failed",
					)
				}
			},
		},
		{
			name: "migration",
			mutate: func(hooks *RuntimeValidationHooksV1) {
				hooks.VerifyMigration = func(
					DataDomain,
					string,
					string,
					string,
				) error {
					return errors.New(
						"migration verification failed",
					)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan, cfg, evidence, hooks :=
				runtimeReadinessFixtureV1(t)

			test.mutate(&hooks)

			if err := ValidateRuntimeReadinessV1(
				plan,
				cfg,
				evidence,
				hooks,
			); err == nil {
				t.Fatalf(
					"%s verifier failure must propagate",
					test.name,
				)
			}
		})
	}
}

func TestRuntimeReadinessRejectsEqualMigrationSchemas(
	t *testing.T,
) {
	plan, cfg, evidence, hooks := runtimeReadinessFixtureV1(t)

	for index, domain := range plan.Domains {
		if domain.MigrationRequired {
			evidence[index].MigrationFromSchema =
				domain.SchemaVersion
			break
		}
	}

	if err := ValidateRuntimeReadinessV1(
		plan,
		cfg,
		evidence,
		hooks,
	); err == nil {
		t.Fatal(
			"equal migration source and target schemas must fail",
		)
	}
}

func runtimeReadinessFixtureV1(
	t *testing.T,
) (
	Plan,
	StorageConfig,
	[]DomainRuntimeEvidenceV1,
	RuntimeValidationHooksV1,
) {
	t.Helper()

	plan := DefaultPlanV1()
	root := t.TempDir()

	cfg := StorageConfig{
		RootDir: filepath.Join(root, "storage"),
		Network: NetworkTestnet,
	}

	backupRoot := filepath.Join(root, "backups")
	restoreRoot := filepath.Join(root, "restore-dry-run")

	evidence := make(
		[]DomainRuntimeEvidenceV1,
		0,
		len(plan.Domains),
	)

	for _, domain := range plan.Domains {
		domainPath, err := EnsureDomainPath(cfg, domain)
		if err != nil {
			t.Fatalf(
				"ensure domain path %s: %v",
				domain.Name,
				err,
			)
		}

		dataPath := filepath.Join(
			domainPath,
			"data.bin",
		)

		data := []byte(
			"encrypted:v1:runtime-data:" +
				domain.Name +
				":" +
				domain.SchemaVersion,
		)

		if err := os.WriteFile(
			dataPath,
			data,
			0o600,
		); err != nil {
			t.Fatalf(
				"write domain data %s: %v",
				domain.Name,
				err,
			)
		}

		backupPath := filepath.Join(
			backupRoot,
			domain.Name,
			domain.SchemaVersion,
			"data.bin",
		)

		manifest, err := BackupFileV1(
			dataPath,
			backupPath,
		)
		if err != nil {
			t.Fatalf(
				"backup domain %s: %v",
				domain.Name,
				err,
			)
		}

		evidence = append(
			evidence,
			DomainRuntimeEvidenceV1{
				DomainName:     domain.Name,
				DomainPath:     domainPath,
				DataPath:       dataPath,
				ObservedSchema: domain.SchemaVersion,
				BackupManifest: manifest,
				RestoreTarget: filepath.Join(
					restoreRoot,
					domain.Name,
					domain.SchemaVersion,
					"data.bin",
				),
				MigrationFromSchema: "legacy_" +
					domain.SchemaVersion,
			},
		)
	}

	hooks := RuntimeValidationHooksV1{
		VerifySchema: func(
			domain DataDomain,
			dataPath string,
			observedSchema string,
		) error {
			if domain.SchemaVersion != observedSchema {
				return errors.New("schema mismatch")
			}

			info, err := os.Stat(dataPath)
			if err != nil {
				return err
			}

			if !info.Mode().IsRegular() {
				return errors.New(
					"schema data is not a regular file",
				)
			}

			return nil
		},
		VerifyEncryption: func(
			_ DataDomain,
			dataPath string,
		) error {
			data, err := os.ReadFile(dataPath)
			if err != nil {
				return err
			}

			if !bytes.HasPrefix(
				data,
				[]byte("encrypted:v1:"),
			) {
				return errors.New(
					"persistence data lacks encryption format marker",
				)
			}

			return nil
		},
		VerifyAppendOnly: func(
			_ DataDomain,
			dataPath string,
		) error {
			info, err := os.Stat(dataPath)
			if err != nil {
				return err
			}

			if !info.Mode().IsRegular() {
				return errors.New(
					"append-only data is not a regular file",
				)
			}

			return nil
		},
		VerifyMigration: func(
			_ DataDomain,
			dataPath string,
			fromSchema string,
			toSchema string,
		) error {
			if fromSchema == "" || toSchema == "" {
				return errors.New(
					"migration schemas required",
				)
			}

			if fromSchema == toSchema {
				return errors.New(
					"migration schemas must differ",
				)
			}

			_, err := os.Stat(dataPath)
			return err
		},
	}

	return plan, cfg, evidence, hooks
}
