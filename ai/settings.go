package ai

import (
	"os"
	"strings"
	"sync"
)

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

// DIKKAT: Projede zaten bir Enabled() vardıysa, onu kaldırıp
// bu dosyadaki Enabled() fonksiyonunu kullanmalısın.

// Enabled, AI motorunun gerçekten analiz yapıp yapmayacağını söyler.
// Hem global enable flag'e, hem de level'in OFF olup olmamasına bakar.

// Level, geçerli AI seviyesini döner.
func Level() AILevel {
	cfgOnce.Do(loadAIConfig)
	return aiCfg.level
}
