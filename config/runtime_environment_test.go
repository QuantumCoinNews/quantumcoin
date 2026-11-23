package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSharedEnvironmentConformanceV1(t *testing.T) {
	raw, err := os.ReadFile("runtime-environment.cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases []struct {
			Name        string            `json:"name"`
			Service     string            `json:"service"`
			Environment map[string]string `json:"environment"`
			Errors      map[string]string `json:"errors"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Cases) < 60 {
		t.Fatal("incomplete conformance fixture")
	}
	for _, c := range fixture.Cases {
		t.Run(c.Name, func(t *testing.T) {
			got := ValidateRuntimeEnvironmentV1(c.Environment, c.Service)
			if !reflect.DeepEqual(got, c.Errors) {
				t.Fatalf("diagnostic mismatch: got=%v want=%v", got, c.Errors)
			}
		})
	}
}

func TestSharedEnvironmentRejectsBeforeConfigPublicationV1(t *testing.T) {
	before := Current()
	t.Setenv("QC_MINING_DELAY_MS", "test-only-invalid-secret-marker")
	_, err := Load(filepath.Join(t.TempDir(), "absent.json"))
	if err == nil || !strings.Contains(err.Error(), "QC_MINING_DELAY_MS") || strings.Contains(err.Error(), "test-only-invalid-secret-marker") {
		t.Fatal("expected redacted startup rejection")
	}
	if !reflect.DeepEqual(Current(), before) {
		t.Fatal("invalid environment published configuration")
	}
}

func TestSharedEnvironmentDoesNotMutateInputV1(t *testing.T) {
	env := map[string]string{"QC_API_TOKEN": "test-only-opaque-token", "QC_TESTNET": "true"}
	before := map[string]string{}
	for key, value := range env {
		before[key] = value
	}
	ValidateRuntimeEnvironmentV1(env, "node")
	if !reflect.DeepEqual(env, before) {
		t.Fatal("validator changed environment")
	}
}

func TestSharedConfigRejectsDuplicateFileKeysBeforePublicationV1(t *testing.T) {
	for _, raw := range []string{`{"http_port":":8082","http_port":":18082"}`, `{"miner":{"address":"test-only-a","address":"test-only-b"}}`} {
		t.Run(raw[:12], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			before := Current()
			_, err := Load(path)
			if err == nil || !strings.Contains(err.Error(), "duplicate configuration key") {
				t.Fatal("duplicate key accepted")
			}
			if !reflect.DeepEqual(Current(), before) {
				t.Fatal("duplicate config published")
			}
		})
	}
}

func TestSharedConfigRequiresRootObjectAndCoherentEffectivePortV1(t *testing.T) {
	for _, raw := range []string{`null`, `[]`, `42`} {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "JSON object") {
			t.Fatal("non-object config accepted")
		}
	}
	t.Setenv("QC_API_LISTEN", "127.0.0.1:18082")
	t.Setenv("QC_HTTP_PORT", "")
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"http_port":":8082"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "QC_API_LISTEN:http-port-conflict") {
		t.Fatal("conflicting effective file port accepted")
	}
}
