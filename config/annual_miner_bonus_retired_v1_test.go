package config

import "testing"

func TestAnnualMinerBonusRetiredV1(t *testing.T) {
	for _, raw := range []string{"", "100", "250", "9223372036854775807", "invalid"} {
		t.Run("legacy_env_"+raw, func(t *testing.T) {
			t.Setenv("QC_ANNUAL_BONUS_QC", raw)
			cfg := Default()
			applyEnv(cfg)
			if got := AnnualBonusPerYearQC; got != 0 {
				t.Fatalf("QMA0047_RETIREMENT_RED: retired annual mining bonus = %d QC with legacy ENV %q; want 0", got, raw)
			}
		})
	}
}
