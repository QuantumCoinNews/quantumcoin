package persistence

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupFileRoundTrip(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "chain.gob")
	backup := filepath.Join(root, "backups", "chain.gob")

	data := []byte("chain-data")
	if err := os.WriteFile(source, data, 0o600); err != nil {
		t.Fatal(err)
	}

	manifest, err := BackupFileV1(source, backup)
	if err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	if err := VerifyBackupFileV1(
		backup,
		manifest.SHA256,
		manifest.Size,
	); err != nil {
		t.Fatalf("backup verification failed: %v", err)
	}
}

func TestBackupFileRefusesOverwrite(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "chain.gob")
	backup := filepath.Join(root, "backup.gob")

	if err := os.WriteFile(source, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(backup, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := BackupFileV1(source, backup); err == nil {
		t.Fatal("backup must refuse overwrite")
	}
}

func TestRestoreFileDryRunDoesNotWrite(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "chain.gob")
	backup := filepath.Join(root, "backup.gob")
	target := filepath.Join(root, "restore", "chain.gob")

	data := []byte("chain-data")
	if err := os.WriteFile(source, data, 0o600); err != nil {
		t.Fatal(err)
	}

	manifest, err := BackupFileV1(source, backup)
	if err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	if _, err := RestoreFileDryRunV1(
		backup,
		manifest.SHA256,
		manifest.Size,
		target,
	); err != nil {
		t.Fatalf("restore dry-run failed: %v", err)
	}

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("dry-run restore must not create target")
	}
}
