package ai

import (
	"os"
	"strings"
	"sync"
)

// AILevel documents the QuantumCoin component used by this module.
// AI analiz seviyesi
type AILevel string

const (
	AILevelOff        AILevel = "off"
	AILevelLow        AILevel = "low"
	AILevelNormal     AILevel = "normal"
	AILevelAggressive AILevel = "aggressive"
)

var (
	cfgOnce sync.Once
	cfgMu   sync.RWMutex

	aiCfg struct {
		enabled bool
		level   AILevel
	}
)

// ENV'den ayarları bir kez yükler
// QC_AI_ENABLED: "0", "false" => kapalı; boşsa varsayılan: açık
// QC_AI_LEVEL: "off" | "low" | "normal" | "aggressive" (boşsa "normal")
func loadAIConfig() {
	enabled := true
	if v := strings.TrimSpace(os.Getenv("QC_AI_ENABLED")); v != "" {
		l := strings.ToLower(v)
		if l == "0" || l == "false" || l == "off" {
			enabled = false
		}
	}

	level := AILevelNormal
	if v := strings.TrimSpace(os.Getenv("QC_AI_LEVEL")); v != "" {
		switch strings.ToLower(v) {
		case "off":
			level = AILevelOff
		case "low":
			level = AILevelLow
		case "normal":
			level = AILevelNormal
		case "aggressive":
			level = AILevelAggressive
		default:
			level = AILevelNormal
		}
	}

	aiCfg.enabled = enabled
	aiCfg.level = level
}

// Environment is read once, before any reader or runtime override. Off always
// disables analysis, including after SetEnabled(true). No setting affects admission.
func configSnapshot() (bool, AILevel) {
	cfgOnce.Do(loadAIConfig)
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	return aiCfg.enabled && aiCfg.level != AILevelOff, aiCfg.level
}

func Level() AILevel {
	_, level := configSnapshot()
	return level
}
