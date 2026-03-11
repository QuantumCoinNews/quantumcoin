package ai

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// JSON içine yazacağımız birleşik state
type aiStateFile struct {
	Telemetry TelemetrySnapshot `json:"telemetry"`
	Bonus     AIBonus           `json:"bonus"`
	Alerts    []AIAlert         `json:"alerts"`
}

func aiStatePath() string {
	dir := os.Getenv("QC_NODE_DIR")
	if dir == "" {
		// fallback: mevcut çalışma dizini
		if wd, err := os.Getwd(); err == nil {
			dir = wd
		} else {
			dir = "."
		}
	}
	return filepath.Join(dir, "ai_state.json")
}

// Node açılırken çağırabileceğin fonksiyon.
// Hata dönerse fatal değil; istersen log'lar, yok sayarsın.
func LoadState() error {
	path := aiStatePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // ilk açılış gibi düşün
		}
		return err
	}
	var st aiStateFile
	if err := json.Unmarshal(data, &st); err != nil {
		return err
	}

	// snapshot'ları içeri bas
	setTelemetryFromSnapshot(st.Telemetry)
	setBonusFromSnapshot(st.Bonus)
	setAlertsFromSnapshot(st.Alerts)

	return nil
}

// Node kapanırken veya belli aralıklarla çağırabileceğin fonksiyon.
func SaveState() error {
	// mevcut snapshot'ları çek
	st := aiStateFile{
		Telemetry: GetTelemetrySnapshot(),
		Bonus:     GetBonusSnapshot(),
		Alerts:    GetAlertsSnapshot(),
	}

	path := aiStatePath()
	tmp := path + ".tmp"

	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
