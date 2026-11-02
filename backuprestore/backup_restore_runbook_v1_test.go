package backuprestore

import "testing"

func TestDefaultBackupRestoreRunbookValid(t *testing.T) {
	runbook := DefaultRunbookV1()

	if err := ValidateRunbookV1(runbook); err != nil {
		t.Fatalf("default runbook should validate: %v", err)
	}
	if err := HasRequiredArtifactsV1(runbook); err != nil {
		t.Fatalf("default runbook missing artifact: %v", err)
	}
}

func TestBackupRestoreSafeBoundary(t *testing.T) {
	runbook := DefaultRunbookV1()
	for _, k := range []string{
		"doesNotRunLiveRestore",
		"doesNotDeleteData",
		"doesNotOverwriteData",
		"doesNotMint",
		"doesNotSendFunds",
		"doesNotChangeBalances",
		"doesNotChangeChain",
		"doesNotChangeSupply",
		"requiresManifest",
		"requiresHashCheck",
		"requiresDryRun",
		"requiresAdminApproval",
	} {
		if !runbook.SafeBoundary[k] {
			t.Fatalf("safe boundary missing: %s", k)
		}
	}
}

func TestBackupRestoreRejectsLiveRestoreAllowed(t *testing.T) {
	runbook := DefaultRunbookV1()
	runbook.Artifacts[0].LiveRestoreForbidden = false

	if err := ValidateRunbookV1(runbook); err == nil {
		t.Fatalf("live restore allowed must fail")
	}
}

func TestBackupRestoreRejectsMissingHash(t *testing.T) {
	runbook := DefaultRunbookV1()
	runbook.Artifacts[0].RequiresHash = false

	if err := ValidateRunbookV1(runbook); err == nil {
		t.Fatalf("missing hash requirement must fail")
	}
}

func TestBackupRestoreRejectsDuplicateArtifact(t *testing.T) {
	runbook := DefaultRunbookV1()
	runbook.Artifacts = append(runbook.Artifacts, runbook.Artifacts[0])

	if err := ValidateRunbookV1(runbook); err == nil {
		t.Fatalf("duplicate artifact must fail")
	}
}
