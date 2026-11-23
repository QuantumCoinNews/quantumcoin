package ai

func Enabled() bool {
	enabled, _ := configSnapshot()
	return enabled
}

func SetEnabled(v bool) {
	cfgOnce.Do(loadAIConfig)
	cfgMu.Lock()
	aiCfg.enabled = v
	cfgMu.Unlock()
}
