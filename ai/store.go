package ai

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const maxStoreBytes = 4 << 20

var storeMu sync.Mutex

func boundedText(s string) string {
	if len(s) > 1024 {
		return s[:1024]
	}
	return s
}
func readBoundedFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maxStoreBytes+1))
	if err == nil && len(b) > maxStoreBytes {
		err = errors.New("AI record exceeds size limit")
	}
	return b, err
}

// Caller holds storeMu. Unique temporary files, sync and OS replacement avoid
// partial JSON and collisions between concurrent writers in this process.
func writeJSONAtomic(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if len(b) > maxStoreBytes {
		return errors.New("AI record exceeds size limit")
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".ai-record-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(b)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return replaceRecord(name, path)
}

type PersistenceStatus struct {
	Failures      uint64 `json:"failures"`
	LastFailureAt int64  `json:"last_failure_at"`
	LastError     string `json:"last_error,omitempty"`
}

var statusMu sync.RWMutex
var persistenceStatus PersistenceStatus

// RecordPersistenceFailure Failures remain observable in memory even if the disk is unavailable.
func RecordPersistenceFailure(operation string, err error) {
	if err == nil {
		return
	}
	message := boundedText(fmt.Sprintf("%s: %v", operation, err))
	statusMu.Lock()
	persistenceStatus.Failures++
	persistenceStatus.LastFailureAt = time.Now().Unix()
	persistenceStatus.LastError = message
	statusMu.Unlock()
	log.Printf("AI advisory persistence: %s", message)
	PushAlert(AlertError, "AI_PERSISTENCE_FAILED", message, "advisory-storage")
}
func GetPersistenceStatus() PersistenceStatus {
	statusMu.RLock()
	defer statusMu.RUnlock()
	return persistenceStatus
}
