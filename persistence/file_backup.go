package persistence

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type BackupManifestV1 struct {
	Version string `json:"version"`
	Source  string `json:"source"`
	Backup  string `json:"backup"`
	Size    int64  `json:"size"`
	SHA256  string `json:"sha256"`
}

const BackupManifestVersionV1 = "QC_BACKUP_MANIFEST_V1"

func BackupFileV1(sourcePath, backupPath string) (BackupManifestV1, error) {
	var manifest BackupManifestV1

	if sourcePath == "" || backupPath == "" {
		return manifest, errors.New("source and backup paths required")
	}

	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return manifest, fmt.Errorf("read backup source: %w", err)
	}

	if _, err := os.Stat(backupPath); err == nil {
		return manifest, errors.New("backup target already exists")
	} else if !os.IsNotExist(err) {
		return manifest, fmt.Errorf("inspect backup target: %w", err)
	}

	parent := filepath.Dir(backupPath)
	if parent != "." && parent != "" {
		if err := os.MkdirAll(parent, 0o700); err != nil {
			return manifest, fmt.Errorf("create backup directory: %w", err)
		}
	}

	sum := sha256.Sum256(source)

	if err := os.WriteFile(backupPath, source, 0o600); err != nil {
		return manifest, fmt.Errorf("write backup: %w", err)
	}

	return BackupManifestV1{
		Version: BackupManifestVersionV1,
		Source:  sourcePath,
		Backup:  backupPath,
		Size:    int64(len(source)),
		SHA256:  hex.EncodeToString(sum[:]),
	}, nil
}

func VerifyBackupFileV1(
	backupPath string,
	expectedSHA256 string,
	expectedSize int64,
) error {
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("read backup: %w", err)
	}

	if expectedSize >= 0 && int64(len(data)) != expectedSize {
		return fmt.Errorf(
			"backup size mismatch: got=%d want=%d",
			len(data),
			expectedSize,
		)
	}

	sum := sha256.Sum256(data)
	actual := hex.EncodeToString(sum[:])

	if actual != expectedSHA256 {
		return fmt.Errorf(
			"backup hash mismatch: got=%s want=%s",
			actual,
			expectedSHA256,
		)
	}

	return nil
}
func RestoreFileDryRunV1(
	backupPath string,
	expectedSHA256 string,
	expectedSize int64,
	targetPath string,
) (BackupManifestV1, error) {
	if backupPath == "" || targetPath == "" {
		return BackupManifestV1{}, errors.New(
			"backup and target paths required",
		)
	}

	if err := VerifyBackupFileV1(
		backupPath,
		expectedSHA256,
		expectedSize,
	); err != nil {
		return BackupManifestV1{}, err
	}

	if _, err := os.Stat(targetPath); err == nil {
		return BackupManifestV1{}, errors.New(
			"restore target already exists",
		)
	} else if !os.IsNotExist(err) {
		return BackupManifestV1{}, fmt.Errorf(
			"inspect restore target: %w",
			err,
		)
	}

	return BackupManifestV1{
		Version: BackupManifestVersionV1,
		Source:  backupPath,
		Backup:  targetPath,
		Size:    expectedSize,
		SHA256:  expectedSHA256,
	}, nil
}
