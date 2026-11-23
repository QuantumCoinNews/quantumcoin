package token

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
)

func TestInitializedTokenStateRejectsJSONReplacement(t *testing.T) {
	owner := canonicalAddressFixture("state-owner")
	attacker := canonicalAddressFixture("state-attacker")
	cases := []struct {
		name        string
		live        any
		replacement any
	}{
		{"Token", New("OLD", "Original", 8, 100, owner), New("EVIL", "Replacement", 0, 999, attacker)},
		{"QC20", NewQC20TokenWithOwner("Original", "OLD", 100, 8, owner), NewQC20TokenWithOwner("Replacement", "EVIL", 999, 0, attacker)},
		{"QC721", NewQC721TokenWithOwner("Original", "OLD", owner), NewQC721TokenWithOwner("Replacement", "EVIL", attacker)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Seed non-empty state so replacement cannot erase balances,
			// delegated spending rights, ownership or consumed replay nonces.
			switch live := tc.live.(type) {
			case *Token:
				live.balances[owner] = 7
				live.totalSupply = 7
				live.authorizationNonces[owner] = 4
			case *QC20Token:
				live.allowances[owner] = map[string]uint64{attacker: 3}
				live.authorizationNonces[owner] = 4
			case *QC721Token:
				live.owners[1] = owner
				live.tokenURIs[1] = "ipfs://original"
				live.authorizationNonces[owner] = 4
			}
			before, err := json.Marshal(tc.live)
			if err != nil {
				t.Fatal(err)
			}
			payload, err := json.Marshal(tc.replacement)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(payload, tc.live); err == nil {
				t.Fatal("JSON replaced live token state without an authorized operation")
			}
			after, err := json.Marshal(tc.live)
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) {
				t.Fatal("rejected JSON changed state")
			}
		})
	}
}

func TestTokenJSONInitializationIsSingleUseUnderConcurrency(t *testing.T) {
	owner := canonicalAddressFixture("state-owner")
	cases := []struct {
		name   string
		empty  any
		source any
	}{
		{"Token", new(Token), New("T", "Test", 8, 100, owner)},
		{"QC20", new(QC20Token), NewQC20TokenWithOwner("Test", "T", 100, 8, owner)},
		{"QC721", new(QC721Token), NewQC721TokenWithOwner("Test", "T", owner)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := json.Marshal(tc.source)
			if err != nil {
				t.Fatal(err)
			}
			var accepted atomic.Int32
			var wg sync.WaitGroup
			for range 20 {
				wg.Go(func() {
					if json.Unmarshal(payload, tc.empty) == nil {
						accepted.Add(1)
					}
				})
			}
			wg.Wait()
			if accepted.Load() != 1 {
				t.Fatalf("initializations = %d, want 1", accepted.Load())
			}
		})
	}
}

func TestRegistryJSONCannotReplaceLiveEntries(t *testing.T) {
	registry := registryWithPreservedTokenV1(t)
	if err := json.Unmarshal([]byte(`{"tokens":{}}`), registry); err == nil {
		t.Fatal("untrusted JSON bypassed Registry.Load integrity boundary")
	}
	registered, ok := registry.Get("KEEP")
	if !ok {
		t.Fatal("registered token missing")
	}
	payload, err := json.Marshal(New("EVIL", "Replacement", 8, 999, canonicalAddressFixture("state-attacker")))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, registered); err == nil {
		t.Fatal("registry lookup allowed JSON replacement of registered token")
	}
	assertPreservedRegistryV1(t, registry)
}

func TestRegistryConcurrentReadRegistrationAndSerialization(t *testing.T) {
	registry := registryWithPreservedTokenV1(t)
	owner := canonicalAddressFixture("state-owner")
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 20 {
				if _, err := json.Marshal(registry); err != nil {
					t.Error(err)
				}
				if tok, ok := registry.Get("KEEP"); !ok || tok.TotalSupply() != 25 {
					t.Error("concurrent registry read lost original state")
				}
				if err := registry.Register(New("KEEP", "Duplicate", 8, 1000, owner)); err == nil {
					t.Error("duplicate registration replaced original entry")
				}
			}
		})
	}
	wg.Wait()
	assertPreservedRegistryV1(t, registry)
}
