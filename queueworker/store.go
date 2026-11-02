package queueworker

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const QueueSnapshotVersionV1 = "QC_QUEUE_SNAPSHOT_V1"

type QueueSnapshotV1 struct {
	Version       string            `json:"version"`
	Jobs          map[string]Job    `json:"jobs"`
	Order         []string          `json:"order"`
	Idempotency   map[string]string `json:"idempotency"`
	Audit         []AuditEvent      `json:"audit"`
	AuditSequence uint64            `json:"auditSequence"`
}

type StateStore interface {
	Load() (QueueSnapshotV1, bool, error)
	Save(snapshot QueueSnapshotV1) error
}

type FileStore struct {
	mu       sync.Mutex
	path     string
	lockPath string
}

func NewFileStore(path string) (*FileStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("queue store path required")
	}

	if !filepath.IsAbs(path) {
		return nil, errors.New(
			"queue store path must be absolute",
		)
	}

	return &FileStore{
		path:     filepath.Clean(path),
		lockPath: filepath.Clean(path) + ".lock",
	}, nil
}
func (store *FileStore) Path() string {
	if store == nil {
		return ""
	}

	return store.path
}

func (store *FileStore) Load() (
	QueueSnapshotV1,
	bool,
	error,
) {
	if store == nil {
		return QueueSnapshotV1{}, false, errors.New(
			"nil queue file store",
		)
	}
	store.mu.Lock()
	defer store.mu.Unlock()

	release, err := acquireProcessLockV1(store.lockPath)
	if err != nil {
		return QueueSnapshotV1{}, false, err
	}
	defer release()

	snapshot, exists, err := loadQueueSnapshotFileV1(store.path)
	if err == nil && exists {
		return snapshot, true, nil
	}

	previousPath := store.path + ".previous"
	previousSnapshot, previousExists, previousErr :=
		loadQueueSnapshotFileV1(previousPath)

	if previousErr == nil && previousExists {
		// Kurtarılan dosyayı kaybetmemek için önce mevcut bozuk
		// snapshot'ı geri alınabilir .corrupt dosyasına taşı.
		if _, statErr := os.Stat(store.path); statErr == nil {
			corruptPath := store.path + ".corrupt"
			_ = os.Remove(corruptPath)
			_ = os.Rename(store.path, corruptPath)
		}

		if renameErr := os.Rename(previousPath, store.path); renameErr != nil {
			return QueueSnapshotV1{}, false, fmt.Errorf(
				"restore previous queue snapshot: %w",
				renameErr,
			)
		}

		return previousSnapshot, true, nil
	}

	if err != nil && !os.IsNotExist(err) {
		if previousErr != nil && !os.IsNotExist(previousErr) {
			return QueueSnapshotV1{}, false, fmt.Errorf(
				"read queue snapshot: %w; previous snapshot: %v",
				err,
				previousErr,
			)
		}

		return QueueSnapshotV1{}, false, err
	}

	if previousErr != nil && !os.IsNotExist(previousErr) {
		return QueueSnapshotV1{}, false, fmt.Errorf(
			"read previous queue snapshot: %w",
			previousErr,
		)
	}

	return QueueSnapshotV1{}, false, nil
}

func loadQueueSnapshotFileV1(
	path string,
) (QueueSnapshotV1, bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return QueueSnapshotV1{}, false, nil
	}
	if err != nil {
		return QueueSnapshotV1{}, false, fmt.Errorf(
			"read queue snapshot %s: %w",
			path,
			err,
		)
	}

	var snapshot QueueSnapshotV1
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return QueueSnapshotV1{}, false, fmt.Errorf(
			"decode queue snapshot %s: %w",
			path,
			err,
		)
	}

	if err := ValidateQueueSnapshotV1(snapshot); err != nil {
		return QueueSnapshotV1{}, false, fmt.Errorf(
			"validate queue snapshot %s: %w",
			path,
			err,
		)
	}

	return cloneQueueSnapshotV1(snapshot), true, nil
}
func (store *FileStore) Save(
	snapshot QueueSnapshotV1,
) error {
	if store == nil {
		return errors.New("nil queue file store")
	}

	if err := ValidateQueueSnapshotV1(snapshot); err != nil {
		return fmt.Errorf(
			"validate queue snapshot before save: %w",
			err,
		)
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	release, err := acquireProcessLockV1(store.lockPath)
	if err != nil {
		return err
	}
	defer release()

	parent := filepath.Dir(store.path)
	if parent == "." || parent == "" {
		return errors.New(
			"queue store parent directory required",
		)
	}

	if err := os.MkdirAll(parent, 0o700); err != nil {
		return fmt.Errorf(
			"create queue store directory: %w",
			err,
		)
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf(
			"encode queue snapshot: %w",
			err,
		)
	}
	data = append(data, '\n')

	tempPath := store.path + ".tmp"
	backupPath := store.path + ".previous"

	if err := writeSyncedFileV1(
		tempPath,
		data,
	); err != nil {
		return err
	}

	_ = os.Remove(backupPath)

	hadCurrent := false
	if _, err := os.Stat(store.path); err == nil {
		hadCurrent = true

		if err := os.Rename(
			store.path,
			backupPath,
		); err != nil {
			_ = os.Remove(tempPath)
			return fmt.Errorf(
				"stage previous queue snapshot: %w",
				err,
			)
		}
	} else if !os.IsNotExist(err) {
		_ = os.Remove(tempPath)
		return fmt.Errorf(
			"inspect current queue snapshot: %w",
			err,
		)
	}

	if err := os.Rename(
		tempPath,
		store.path,
	); err != nil {
		if hadCurrent {
			_ = os.Rename(
				backupPath,
				store.path,
			)
		}

		_ = os.Remove(tempPath)

		return fmt.Errorf(
			"commit queue snapshot: %w",
			err,
		)
	}

	_ = os.Remove(backupPath)

	return nil
}

func ValidateQueueSnapshotV1(
	snapshot QueueSnapshotV1,
) error {
	if snapshot.Version != QueueSnapshotVersionV1 {
		return fmt.Errorf(
			"unsupported queue snapshot version: %q",
			snapshot.Version,
		)
	}

	if snapshot.Jobs == nil {
		return errors.New("queue snapshot jobs required")
	}

	if snapshot.Idempotency == nil {
		return errors.New(
			"queue snapshot idempotency index required",
		)
	}

	if len(snapshot.Order) != len(snapshot.Jobs) {
		return fmt.Errorf(
			"queue snapshot order mismatch: order=%d jobs=%d",
			len(snapshot.Order),
			len(snapshot.Jobs),
		)
	}

	ordered := make(
		map[string]struct{},
		len(snapshot.Order),
	)

	for _, jobID := range snapshot.Order {
		if strings.TrimSpace(jobID) == "" {
			return errors.New(
				"queue snapshot order contains empty job ID",
			)
		}

		if _, exists := ordered[jobID]; exists {
			return fmt.Errorf(
				"queue snapshot order contains duplicate job: %s",
				jobID,
			)
		}

		if _, exists := snapshot.Jobs[jobID]; !exists {
			return fmt.Errorf(
				"queue snapshot order references missing job: %s",
				jobID,
			)
		}

		ordered[jobID] = struct{}{}
	}

	for mapID, job := range snapshot.Jobs {
		if mapID != job.ID {
			return fmt.Errorf(
				"queue snapshot job key mismatch: key=%s job=%s",
				mapID,
				job.ID,
			)
		}

		if err := validateSnapshotJobV1(job); err != nil {
			return fmt.Errorf(
				"invalid snapshot job %s: %w",
				mapID,
				err,
			)
		}

		indexKey := compoundIdempotencyKeyV1(
			job.Queue,
			job.IdempotencyKey,
		)

		indexedJobID, exists := snapshot.Idempotency[indexKey]
		if !exists {
			return fmt.Errorf(
				"missing idempotency index for job: %s",
				job.ID,
			)
		}

		if indexedJobID != job.ID {
			return fmt.Errorf(
				"idempotency index mismatch for job: %s",
				job.ID,
			)
		}
	}

	for key, jobID := range snapshot.Idempotency {
		job, exists := snapshot.Jobs[jobID]
		if !exists {
			return fmt.Errorf(
				"idempotency index references missing job: %s",
				jobID,
			)
		}

		expectedKey := compoundIdempotencyKeyV1(
			job.Queue,
			job.IdempotencyKey,
		)

		if key != expectedKey {
			return fmt.Errorf(
				"invalid idempotency index key for job: %s",
				jobID,
			)
		}
	}

	var previousSequence uint64

	for index, event := range snapshot.Audit {
		if event.Sequence == 0 {
			return fmt.Errorf(
				"audit event %d has zero sequence",
				index,
			)
		}

		if event.Sequence <= previousSequence {
			return fmt.Errorf(
				"audit sequence is not increasing at index %d",
				index,
			)
		}

		if event.Sequence > snapshot.AuditSequence {
			return fmt.Errorf(
				"audit event sequence exceeds snapshot sequence",
			)
		}

		previousSequence = event.Sequence
	}

	if len(snapshot.Audit) > 0 &&
		snapshot.AuditSequence <
			snapshot.Audit[len(snapshot.Audit)-1].Sequence {
		return errors.New(
			"invalid queue snapshot audit sequence",
		)
	}

	return nil
}

func validateSnapshotJobV1(job Job) error {
	if strings.TrimSpace(job.ID) == "" {
		return errors.New("job ID required")
	}

	if strings.TrimSpace(job.Queue) == "" {
		return errors.New("job queue required")
	}

	if strings.TrimSpace(job.IdempotencyKey) == "" {
		return errors.New(
			"job idempotency key required",
		)
	}

	if !json.Valid(job.Payload) {
		return errors.New(
			"job payload must contain valid JSON",
		)
	}

	if !IsKnownJobStatus(job.Status) {
		return fmt.Errorf(
			"unknown job status: %q",
			job.Status,
		)
	}

	if job.MaxAttempts < 1 {
		return errors.New(
			"job maximum attempts must be positive",
		)
	}

	if job.Attempts < 0 ||
		job.Attempts > job.MaxAttempts {
		return errors.New(
			"invalid job attempt count",
		)
	}

	if job.CreatedAt <= 0 ||
		job.UpdatedAt <= 0 {
		return errors.New(
			"job timestamps required",
		)
	}

	switch job.Status {
	case JobStatusLeased:
		if strings.TrimSpace(job.LeaseOwner) == "" ||
			job.LeaseUntil <= 0 {
			return errors.New(
				"leased job requires owner and expiry",
			)
		}

	default:
		if job.LeaseOwner != "" ||
			job.LeaseUntil != 0 {
			return errors.New(
				"non-leased job retains lease state",
			)
		}
	}

	if job.RequiresApproval &&
		job.ApprovedBy == "" &&
		job.ApprovedAt != 0 {
		return errors.New(
			"unapproved job has approval timestamp",
		)
	}

	if job.ApprovedBy != "" &&
		job.ApprovedAt <= 0 {
		return errors.New(
			"approved job requires approval timestamp",
		)
	}

	return nil
}

func cloneQueueSnapshotV1(
	snapshot QueueSnapshotV1,
) QueueSnapshotV1 {
	result := QueueSnapshotV1{
		Version:       snapshot.Version,
		Jobs:          make(map[string]Job, len(snapshot.Jobs)),
		Order:         append([]string(nil), snapshot.Order...),
		Idempotency:   make(map[string]string, len(snapshot.Idempotency)),
		Audit:         append([]AuditEvent(nil), snapshot.Audit...),
		AuditSequence: snapshot.AuditSequence,
	}

	for jobID, job := range snapshot.Jobs {
		cloned := job
		cloned.Payload = append(
			json.RawMessage(nil),
			job.Payload...,
		)
		result.Jobs[jobID] = cloned
	}

	for key, jobID := range snapshot.Idempotency {
		result.Idempotency[key] = jobID
	}

	return result
}

func writeSyncedFileV1(
	path string,
	data []byte,
) error {
	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
		0o600,
	)
	if err != nil {
		return fmt.Errorf(
			"open queue snapshot temp file: %w",
			err,
		)
	}

	writeErr := error(nil)

	if _, err := file.Write(data); err != nil {
		writeErr = fmt.Errorf(
			"write queue snapshot temp file: %w",
			err,
		)
	} else if err := file.Sync(); err != nil {
		writeErr = fmt.Errorf(
			"sync queue snapshot temp file: %w",
			err,
		)
	}

	closeErr := file.Close()

	if writeErr != nil {
		_ = os.Remove(path)
		return writeErr
	}

	if closeErr != nil {
		_ = os.Remove(path)
		return fmt.Errorf(
			"close queue snapshot temp file: %w",
			closeErr,
		)
	}

	return nil
}

const (
	queueStoreLockRetryV1 = 20 * time.Millisecond
	queueStoreLockWaitV1  = 5 * time.Second
)

func acquireProcessLockV1(
	path string,
) (func(), error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("queue store lock path required")
	}

	parent := filepath.Dir(path)
	if parent == "." || parent == "" {
		return nil, errors.New("queue store lock parent directory required")
	}

	if err := os.MkdirAll(parent, 0o700); err != nil {
		return nil, fmt.Errorf(
			"create queue store lock directory: %w",
			err,
		)
	}

	deadline := time.Now().Add(queueStoreLockWaitV1)

	for {
		file, err := os.OpenFile(
			path,
			os.O_CREATE|os.O_EXCL|os.O_WRONLY,
			0o600,
		)
		if err == nil {
			_ = file.Close()

			return func() {
				_ = os.Remove(path)
			}, nil
		}

		if !os.IsExist(err) {
			return nil, fmt.Errorf(
				"create queue store process lock: %w",
				err,
			)
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf(
				"queue store process lock timeout: %w",
				err,
			)
		}

		time.Sleep(queueStoreLockRetryV1)
	}
}
