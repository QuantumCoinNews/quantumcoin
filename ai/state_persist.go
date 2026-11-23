package ai

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Legacy bonus JSON is ignored on load and omitted on save.
type aiStateFile struct {
	Telemetry TelemetrySnapshot `json:"telemetry"`
	Alerts    []AIAlert         `json:"alerts"`
}

func aiStateDir() string {
	if dir := os.Getenv("QC_NODE_DIR"); dir != "" {
		return dir
	}
	return "."
}
func LoadState() error { return LoadStateFrom(aiStateDir()) }
func SaveState() error { return SaveStateTo(aiStateDir()) }

// LoadStateFrom is read-only. Oversize/corrupt files leave live state intact.
func LoadStateFrom(dir string) error {
	storeMu.Lock()
	defer storeMu.Unlock()
	data, err := readBoundedFile(filepath.Join(dir, "ai_state.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var st aiStateFile
	if err = json.Unmarshal(data, &st); err != nil {
		return err
	}
	if !validTelemetry(st.Telemetry) {
		return errors.New("invalid persisted AI telemetry")
	}
	setTelemetryFromSnapshot(st.Telemetry)
	setAlertsFromSnapshot(st.Alerts)
	return nil
}
func SaveStateTo(dir string) error {
	storeMu.Lock()
	defer storeMu.Unlock()
	unlock, err := lockRecordDirectory(dir)
	if err != nil {
		return err
	}
	defer unlock()
	return writeJSONAtomic(filepath.Join(dir, "ai_state.json"), aiStateFile{Telemetry: GetTelemetrySnapshot(), Alerts: GetAlertsSnapshot()})
}
