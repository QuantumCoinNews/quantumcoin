package policyexamples

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckedInPoliciesAreGeneratedFromAuthority(t *testing.T) {
	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range files {
		t.Run(name, func(t *testing.T) {
			got, err := os.ReadFile(filepath.Join("..", "configs", name))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(bytes.ReplaceAll(got, []byte("\r\n"), []byte("\n")), want) {
				t.Fatal("fixture drift; run go run ./cmd/policy-export from repository root")
			}
		})
	}
}
func TestSafeLoadTargetsUseServiceSpecificHealthContracts(t *testing.T) {
	b, err := os.ReadFile("../configs/load_test.safe.example.json")
	if err != nil {
		t.Fatal(err)
	}
	var plan struct {
		Targets []struct {
			Name     string `json:"name"`
			Endpoint string `json:"endpoint"`
		} `json:"targets"`
	}
	if err := json.Unmarshal(b, &plan); err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, target := range plan.Targets {
		seen[target.Name] = target.Endpoint
	}
	if seen["qc_node_health_local"] != "/api/health" || seen["telegram_game_health_local"] != "/health" {
		t.Fatal("health route drift", seen)
	}
}
