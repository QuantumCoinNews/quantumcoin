package main

import (
	"os"
	"strings"
	"testing"
)

func TestPersistenceErrorHandlingSourceContract(t *testing.T) {
	checks := []struct {
		path      string
		forbidden []string
		required  []string
	}{
		{
			path: "blockchain/mined_balance.go",
			forbidden: []string{
				`if b, err := json.MarshalIndent(m, "", "  "); err == nil`,
				"_ = os.WriteFile(tmp, b, 0644)",
				"_ = os.Rename(tmp, p)",
			},
			required: []string{
				"mined balance encode failed",
				"mined balance temp write failed",
				"mined balance replace failed",
			},
		},
		{
			path: "miner/miner.go",
			forbidden: []string{
				`if b, err := json.MarshalIndent(mb, "", "  "); err == nil`,
				"_ = os.WriteFile(tmp, b, 0644)",
				"_ = os.Rename(tmp, p)",
			},
			required: []string{
				"mined balance encode failed",
				"mined balance temp write failed",
				"mined balance replace failed",
			},
		},
		{
			path: "cmd/qc_wallet_create/main.go",
			forbidden: []string{
				`b, _ := json.MarshalIndent(manifest, "", "  ")`,
				"_ = os.WriteFile(",
			},
			required: []string{
				"func failCLI(",
				"func writeRequiredCLIFile(",
				"encode wallet recovery manifest",
			},
		},
		{
			path: "cmd/qc_wallet_restore/main.go",
			forbidden: []string{
				`b, _ := json.MarshalIndent(manifest, "", "  ")`,
				"_ = os.WriteFile(",
			},
			required: []string{
				"func failCLI(",
				"func writeRequiredCLIFile(",
				"encode wallet recovery manifest",
			},
		},
		{
			path: "cmd/wallet/main_fyne.go",
			forbidden: []string{
				`b, _ := json.MarshalIndent(m, "", "  ")`,
				`if b, err := json.MarshalIndent(v, "", "  "); err == nil`,
				"_ = os.WriteFile(balanceCachePath(), b, 0644)",
				"_ = os.WriteFile(filepath.Join(dir, \"wallet_recovery_manifest.json\"), b, 0644)",
				"_ = os.WriteFile(filepath.Join(dir, \"wallet_balance.cache.json\"), b, 0644)",
				"_ = os.WriteFile(filepath.Join(dir, \"bonus_store.json\"), b, 0644)",
			},
			required: []string{
				"func writeRestoredFile(",
				"func writeRestoredJSON(",
				"balance cache encode failed",
				"bonus store encode failed",
			},
		},
		{
			path: "wallet/ensure.go",
			forbidden: []string{
				`data, _ := json.MarshalIndent(w, "", "  ")`,
			},
			required: []string{
				`data, err := json.MarshalIndent(w, "", "  ")`,
			},
		},
	}

	for _, check := range checks {
		src, err := os.ReadFile(check.path)
		if err != nil {
			t.Fatalf(
				"read %s: %v",
				check.path,
				err,
			)
		}

		text := string(src)

		for _, forbidden := range check.forbidden {
			if strings.Contains(text, forbidden) {
				t.Fatalf(
					"%s still contains silent persistence pattern %q",
					check.path,
					forbidden,
				)
			}
		}

		for _, required := range check.required {
			if !strings.Contains(text, required) {
				t.Fatalf(
					"%s missing persistence error contract %q",
					check.path,
					required,
				)
			}
		}
	}
}
