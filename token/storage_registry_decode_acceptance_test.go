package token

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestRegistryLoadRejectsAmbiguousOrIncompleteJSONAtomically(t *testing.T) {
	owner := canonicalAddressFixture("registry-decode-owner")
	state := registryStateJSONV1("NEW", owner, 100, 10, map[string]uint64{owner: 10}, map[string]uint64{owner: 1})
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	tokenJSON := string(raw)
	wrap := func(s string) string { return `{"tokens":{"NEW":` + s + `}}` }
	cases := map[string]string{
		"null root":                 `null`,
		"missing envelope":          `{}`,
		"wrong envelope case":       `{"Tokens":{}}`,
		"unknown envelope field":    `{"tokens":{},"tokenz":{}}`,
		"duplicate envelope":        `{"tokens":{"NEW":` + tokenJSON + `},"tokens":{}}`,
		"case folded envelope":      `{"tokens":{"NEW":` + tokenJSON + `},"Tokens":{}}`,
		"duplicate owner":           wrap(strings.Replace(tokenJSON, `"owner":`, `"owner":"`+owner+`","owner":`, 1)),
		"escaped duplicate owner":   wrap(strings.Replace(tokenJSON, `"owner":`, `"\u006fwner":"`+owner+`","owner":`, 1)),
		"case folded owner":         wrap(strings.TrimSuffix(tokenJSON, "}") + `,"Owner":"` + owner + `"}`),
		"duplicate balance address": wrap(strings.Replace(tokenJSON, `"balances":{"`+owner+`":10}`, `"balances":{"`+owner+`":9,"`+owner+`":10}`, 1)),
		"duplicate nonce address":   wrap(strings.Replace(tokenJSON, `"authorization_nonces":{"`+owner+`":1}`, `"authorization_nonces":{"`+owner+`":99,"`+owner+`":1}`, 1)),
		"missing max supply":        wrap(strings.Replace(tokenJSON, `"max_supply":100,`, "", 1)),
		"null max supply":           wrap(strings.Replace(tokenJSON, `"max_supply":100`, `"max_supply":null`, 1)),
		"unknown token field":       wrap(strings.TrimSuffix(tokenJSON, "}") + `,"typo":1}`),
		"blank name":                wrap(strings.Replace(tokenJSON, `"name":"Registry Integrity Test Token"`, `"name":" "`, 1)),
		"trailing document":         wrap(tokenJSON) + ` {"tokens":{}}`,
	}
	for _, field := range []string{"symbol", "name", "decimals", "max_supply", "total_supply", "owner"} {
		for _, missing := range []bool{false, true} {
			changed := make(map[string]any)
			for key, value := range state {
				changed[key] = value
			}
			label := "null "
			changed[field] = nil
			if missing {
				delete(changed, field)
				label = "missing "
			}
			raw, err := json.Marshal(changed)
			if err != nil {
				t.Fatal(err)
			}
			cases[label+field] = wrap(string(raw))
		}
	}
	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, registryFile)
			if err := os.WriteFile(path, []byte(payload), 0600); err != nil {
				t.Fatal(err)
			}
			registry := registryWithPreservedTokenV1(t)
			before, err := json.Marshal(registry)
			if err != nil {
				t.Fatal(err)
			}
			if err := registry.Load(dir); !errors.Is(err, ErrRegistryLoadIntegrityV1) {
				t.Fatalf("load = %v, want integrity rejection", err)
			}
			after, err := json.Marshal(registry)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Fatal("failed load changed previous registry")
			}
			saved, err := os.ReadFile(path)
			if err != nil || string(saved) != payload {
				t.Fatal("rejected file was changed")
			}
		})
	}
}

func TestRegistryLoadReadFailurePreservesExistingState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, registryFile)
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	registry := registryWithPreservedTokenV1(t)
	if err := registry.Load(dir); err == nil {
		t.Fatal("read failure returned success")
	}
	assertPreservedRegistryV1(t, registry)
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		t.Fatal("failed load changed source directory")
	}
}

func TestRegistryLoadConcurrentReadersNeverObservePartialReplacement(t *testing.T) {
	registry := registryWithPreservedTokenV1(t)
	goodDir, badDir := t.TempDir(), t.TempDir()
	if err := registry.Save(goodDir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badDir, registryFile), []byte(`{"tokens":{"BROKEN":null}}`), 0600); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for worker := range 4 {
		wg.Go(func() {
			for range 6 {
				switch worker {
				case 0:
					if err := registry.Load(goodDir); err != nil {
						t.Error(err)
					}
				case 1:
					if err := registry.Load(badDir); !errors.Is(err, ErrRegistryLoadIntegrityV1) {
						t.Errorf("invalid concurrent load: %v", err)
					}
				default:
					if tok, ok := registry.Get("KEEP"); !ok || tok.TotalSupply() != 25 {
						t.Error("reader observed missing or partial state")
					}
					if _, err := json.Marshal(registry); err != nil {
						t.Error(err)
					}
				}
			}
		})
	}
	wg.Wait()
	assertPreservedRegistryV1(t, registry)
}

func TestRegistryLoadExplicitEmptyAndLegacyFormatsRemainSupported(t *testing.T) {
	for name, payload := range map[string]string{"empty object": `{"tokens":{}}`, "legacy null map": `{"tokens":null}`} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, registryFile), []byte(payload), 0600); err != nil {
				t.Fatal(err)
			}
			registry := registryWithPreservedTokenV1(t)
			if err := registry.Load(dir); err != nil {
				t.Fatal(err)
			}
			if _, ok := registry.Get("KEEP"); ok {
				t.Fatal("explicit empty registry did not replace old state")
			}
			if err := registry.Register(New("AFTER", "After Empty Load", 8, 100, canonicalAddressFixture("after-empty"))); err != nil {
				t.Fatal(err)
			}
		})
	}
}
