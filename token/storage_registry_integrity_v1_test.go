package token

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"quantumcoin/security"
)

func registryStateJSONV1(
	symbol string,
	owner string,
	maxSupply uint64,
	totalSupply uint64,
	balances map[string]uint64,
	nonces map[string]uint64,
) map[string]any {
	return map[string]any{
		"symbol":               symbol,
		"name":                 "Registry Integrity Test Token",
		"decimals":             8,
		"max_supply":           maxSupply,
		"total_supply":         totalSupply,
		"owner":                owner,
		"balances":             balances,
		"authorization_nonces": nonces,
	}
}

func writeRegistryStateJSONV1(
	t *testing.T,
	directory string,
	tokens any,
) {
	t.Helper()

	data, err := json.Marshal(
		map[string]any{
			"tokens": tokens,
		},
	)
	if err != nil {
		t.Fatalf(
			"marshal registry state: %v",
			err,
		)
	}

	if err := os.WriteFile(
		filepath.Join(
			directory,
			registryFile,
		),
		data,
		0o600,
	); err != nil {
		t.Fatalf(
			"write registry state: %v",
			err,
		)
	}
}

func registryWithPreservedTokenV1(
	t *testing.T,
) *Registry {
	t.Helper()

	owner := canonicalAddressFixture(
		"keep-owner",
	)

	registry := NewRegistry()
	subject := New(
		"KEEP",
		"Preserved Token",
		8,
		1000,
		owner,
	)

	if err := subject.Mint(
		owner,
		owner,
		25,
	); err != nil {
		t.Fatalf(
			"mint preserved token: %v",
			err,
		)
	}

	if err := registry.Register(subject); err != nil {
		t.Fatalf(
			"register preserved token: %v",
			err,
		)
	}

	return registry
}

func assertPreservedRegistryV1(
	t *testing.T,
	registry *Registry,
) {
	t.Helper()

	preserved, ok := registry.Get("KEEP")
	if !ok || preserved == nil {
		t.Fatal(
			"previous registry state was not preserved",
		)
	}

	if preserved.TotalSupply() != 25 {
		t.Fatalf(
			"preserved total supply = %d, want 25",
			preserved.TotalSupply(),
		)
	}

	if preserved.BalanceOf(canonicalAddressFixture("keep-owner")) != 25 {
		t.Fatalf(
			"preserved balance = %d, want 25",
			preserved.BalanceOf(canonicalAddressFixture("keep-owner")),
		)
	}
}

func TestRegistryLoadIntegrityV1AcceptsLegacyState(
	t *testing.T,
) {
	directory := t.TempDir()

	legacy := registryStateJSONV1(
		"LEG",
		canonicalAddressFixture("legacy-owner"),
		0,
		100,
		map[string]uint64{
			canonicalAddressFixture("legacy-owner"): 100,
		},
		nil,
	)
	delete(
		legacy,
		"authorization_nonces",
	)

	emptyLegacy := registryStateJSONV1(
		"EMPTY",
		canonicalAddressFixture("legacy-owner"),
		0,
		0,
		nil,
		nil,
	)
	delete(emptyLegacy, "balances")
	delete(
		emptyLegacy,
		"authorization_nonces",
	)

	writeRegistryStateJSONV1(
		t,
		directory,
		map[string]any{
			"LEG":   legacy,
			"EMPTY": emptyLegacy,
		},
	)

	registry := NewRegistry()

	if err := registry.Load(directory); err != nil {
		t.Fatalf(
			"load valid legacy registry: %v",
			err,
		)
	}

	loaded, ok := registry.Get("LEG")
	if !ok || loaded == nil {
		t.Fatal("legacy token was not loaded")
	}

	if loaded.TotalSupply() != 100 {
		t.Fatalf(
			"legacy total supply = %d, want 100",
			loaded.TotalSupply(),
		)
	}

	if loaded.MaxSupply() != 0 {
		t.Fatalf(
			"legacy max supply = %d, want unlimited 0",
			loaded.MaxSupply(),
		)
	}

	if loaded.BalanceOf(canonicalAddressFixture("legacy-owner")) != 100 {
		t.Fatalf(
			"legacy balance = %d, want 100",
			loaded.BalanceOf(canonicalAddressFixture("legacy-owner")),
		)
	}

	emptyLoaded, ok := registry.Get("EMPTY")
	if !ok || emptyLoaded == nil {
		t.Fatal(
			"empty legacy token was not loaded",
		)
	}

	emptyLoaded.mu.Lock()
	defer emptyLoaded.mu.Unlock()

	if emptyLoaded.balances == nil {
		t.Fatal(
			"legacy nil balances map was not repaired",
		)
	}

	if emptyLoaded.authorizationNonces == nil {
		t.Fatal(
			"legacy nil authorization nonce map was not repaired",
		)
	}
}

func TestRegistryLoadIntegrityV1AcceptsNullTokenMap(
	t *testing.T,
) {
	directory := t.TempDir()

	if err := os.WriteFile(
		filepath.Join(
			directory,
			registryFile,
		),
		[]byte(`{"tokens":null}`),
		0o600,
	); err != nil {
		t.Fatalf(
			"write null registry map: %v",
			err,
		)
	}

	registry := NewRegistry()

	if err := registry.Load(directory); err != nil {
		t.Fatalf(
			"load null registry map: %v",
			err,
		)
	}

	if len(registry.Tokens) != 0 {
		t.Fatalf(
			"loaded null registry count = %d, want 0",
			len(registry.Tokens),
		)
	}
}

func TestRegistryLoadIntegrityV1RejectsDuplicateKeysAtomically(
	t *testing.T,
) {
	directory := t.TempDir()

	owner := canonicalAddressFixture(
		"owner",
	)

	duplicate := []byte(`{
"tokens": {
"DUP": {
"symbol": "DUP",
"name": "First",
"decimals": 8,
"max_supply": 1000,
"total_supply": 100,
"owner": "` + owner + `",
"balances": {"` + owner + `": 100}
},
"DUP": {
"symbol": "DUP",
"name": "Second",
"decimals": 8,
"max_supply": 1000,
"total_supply": 100,
"owner": "` + owner + `",
"balances": {"` + owner + `": 100}
}
}
}`)

	if err := os.WriteFile(
		filepath.Join(
			directory,
			registryFile,
		),
		duplicate,
		0o600,
	); err != nil {
		t.Fatalf(
			"write duplicate registry: %v",
			err,
		)
	}

	registry := registryWithPreservedTokenV1(t)
	err := registry.Load(directory)

	if !errors.Is(
		err,
		ErrRegistryLoadIntegrityV1,
	) {
		t.Fatalf(
			"duplicate key error = %v, want integrity error",
			err,
		)
	}

	assertPreservedRegistryV1(
		t,
		registry,
	)
}

func TestRegistryLoadIntegrityV1RejectsInvalidStateAtomically(
	t *testing.T,
) {
	valid := func(symbol string) map[string]any {
		return registryStateJSONV1(
			symbol,
			canonicalAddressFixture("owner"),
			1000,
			100,
			map[string]uint64{
				canonicalAddressFixture("owner"): 100,
			},
			map[string]uint64{
				canonicalAddressFixture("owner"): 1,
			},
		)
	}

	testCases := []struct {
		name   string
		tokens map[string]any
	}{
		{
			name: "nil token",
			tokens: map[string]any{
				"BAD": nil,
			},
		},
		{
			name: "empty registry key",
			tokens: map[string]any{
				"": valid("BAD"),
			},
		},
		{
			name: "whitespace registry key",
			tokens: map[string]any{
				" BAD ": valid("BAD"),
			},
		},
		{
			name: "key symbol mismatch",
			tokens: map[string]any{
				"KEY": valid("SYMBOL"),
			},
		},
		{
			name: "total supply above max supply",
			tokens: map[string]any{
				"BAD": registryStateJSONV1(
					"BAD",
					canonicalAddressFixture("owner"),
					99,
					100,
					map[string]uint64{
						canonicalAddressFixture("owner"): 100,
					},
					nil,
				),
			},
		},
		{
			name: "balance total mismatch",
			tokens: map[string]any{
				"BAD": registryStateJSONV1(
					"BAD",
					canonicalAddressFixture("owner"),
					1000,
					100,
					map[string]uint64{
						canonicalAddressFixture("owner"): 99,
					},
					nil,
				),
			},
		},
		{
			name: "invalid owner",
			tokens: map[string]any{
				"BAD": registryStateJSONV1(
					"BAD",
					" ",
					1000,
					100,
					map[string]uint64{
						canonicalAddressFixture("holder"): 100,
					},
					nil,
				),
			},
		},
		{
			name: "invalid balance address",
			tokens: map[string]any{
				"BAD": registryStateJSONV1(
					"BAD",
					canonicalAddressFixture("owner"),
					1000,
					100,
					map[string]uint64{
						" ": 100,
					},
					nil,
				),
			},
		},
		{
			name: "balance total overflow",
			tokens: map[string]any{
				"BAD": registryStateJSONV1(
					"BAD",
					canonicalAddressFixture("owner"),
					0,
					^uint64(0),
					map[string]uint64{
						canonicalAddressFixture("owner"):  ^uint64(0),
						canonicalAddressFixture("holder"): 1,
					},
					nil,
				),
			},
		},
		{
			name: "invalid nonce caller",
			tokens: map[string]any{
				"BAD": registryStateJSONV1(
					"BAD",
					canonicalAddressFixture("owner"),
					1000,
					100,
					map[string]uint64{
						canonicalAddressFixture("owner"): 100,
					},
					map[string]uint64{
						" ": 1,
					},
				),
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				directory := t.TempDir()

				writeRegistryStateJSONV1(
					t,
					directory,
					testCase.tokens,
				)

				registry :=
					registryWithPreservedTokenV1(t)

				err := registry.Load(directory)

				if !errors.Is(
					err,
					ErrRegistryLoadIntegrityV1,
				) {
					t.Fatalf(
						"load error = %v, want integrity error",
						err,
					)
				}

				assertPreservedRegistryV1(
					t,
					registry,
				)
			},
		)
	}
}

func TestRegistryLoadIntegrityV1MalformedJSONIsAtomic(
	t *testing.T,
) {
	directory := t.TempDir()

	if err := os.WriteFile(
		filepath.Join(
			directory,
			registryFile,
		),
		[]byte(`{"tokens":`),
		0o600,
	); err != nil {
		t.Fatalf(
			"write malformed registry: %v",
			err,
		)
	}

	registry := registryWithPreservedTokenV1(t)

	if err := registry.Load(directory); err == nil {
		t.Fatal(
			"malformed registry load unexpectedly succeeded",
		)
	}

	assertPreservedRegistryV1(
		t,
		registry,
	)
}

func TestRegistryLoadIntegrityV1MissingFilePreservesState(
	t *testing.T,
) {
	directory := t.TempDir()
	registry := registryWithPreservedTokenV1(t)

	if err := registry.Load(directory); err != nil {
		t.Fatalf(
			"missing registry file: %v",
			err,
		)
	}

	assertPreservedRegistryV1(
		t,
		registry,
	)
}

func assertRegistryLoadObservationV1(
	t *testing.T,
	directory string,
	eventType string,
	severity security.EventSeverity,
	outcome string,
	errorClass string,
	tokenCount float64,
) {
	t.Helper()

	eventLog := security.ReadEvents(directory, 10)
	if len(eventLog.Events) != 1 {
		t.Fatalf(
			"registry load event count = %d, want 1",
			len(eventLog.Events),
		)
	}

	event := eventLog.Events[0]

	if event.Type != eventType {
		t.Fatalf(
			"registry load event type = %q, want %q",
			event.Type,
			eventType,
		)
	}

	if event.Severity != severity {
		t.Fatalf(
			"registry load severity = %q, want %q",
			event.Severity,
			severity,
		)
	}

	if event.Source != "token" {
		t.Fatalf(
			"registry load source = %q, want token",
			event.Source,
		)
	}

	if event.Subject != "token_registry" {
		t.Fatalf(
			"registry load subject = %q, want token_registry",
			event.Subject,
		)
	}

	if event.Metadata["model"] !=
		TokenAuthorizationModelLegacyV1 {
		t.Fatalf(
			"registry load model = %v",
			event.Metadata["model"],
		)
	}

	if event.Metadata["operation"] !=
		TokenOperationRegistryLoadV1 {
		t.Fatalf(
			"registry load operation = %v",
			event.Metadata["operation"],
		)
	}

	if event.Metadata["outcome"] != outcome {
		t.Fatalf(
			"registry load outcome = %v, want %q",
			event.Metadata["outcome"],
			outcome,
		)
	}

	if event.Metadata["errorClass"] != errorClass {
		t.Fatalf(
			"registry load error class = %v, want %q",
			event.Metadata["errorClass"],
			errorClass,
		)
	}

	if event.Metadata["tokenCount"] != tokenCount {
		t.Fatalf(
			"registry load token count = %v, want %v",
			event.Metadata["tokenCount"],
			tokenCount,
		)
	}

	if event.Metadata["aiReadOnly"] != true {
		t.Fatal("registry load AI observation is not read-only")
	}

	if event.Metadata["advisoryOnly"] != true {
		t.Fatal("registry load AI observation is not advisory-only")
	}

	if event.Metadata["authorizesOperation"] != false {
		t.Fatal("registry load AI observation authorizes an operation")
	}

	if event.Metadata["changesRegistry"] != false {
		t.Fatal("registry load AI observation can change registry")
	}

	for key := range event.Metadata {
		lowerKey := strings.ToLower(key)

		for _, forbidden := range []string{
			"directory",
			"path",
			"content",
			"private",
			"public",
			"signature",
		} {
			if strings.Contains(lowerKey, forbidden) {
				t.Fatalf(
					"registry load metadata exposes forbidden key %q",
					key,
				)
			}
		}
	}

	raw, err := os.ReadFile(
		filepath.Join(
			directory,
			"security_events.json",
		),
	)
	if err != nil {
		t.Fatalf(
			"read registry load security event: %v",
			err,
		)
	}

	lower := strings.ToLower(string(raw))

	for _, forbidden := range []string{
		"private_key",
		"privatekey",
		"priv_hex",
		"public_key",
		"publickey",
		"signature",
		"token_registry.json",
	} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf(
				"registry load event contains forbidden data %q",
				forbidden,
			)
		}
	}
}

func TestRegistryLoadAIObservabilityV1Accepted(
	t *testing.T,
) {
	directory := t.TempDir()

	writeRegistryStateJSONV1(
		t,
		directory,
		map[string]any{
			"OBS": registryStateJSONV1(
				"OBS",
				canonicalAddressFixture("owner"),
				1000,
				40,
				map[string]uint64{
					canonicalAddressFixture("owner"): 40,
				},
				nil,
			),
		},
	)

	registry := NewRegistry()

	if err := registry.Load(directory); err != nil {
		t.Fatalf(
			"load valid observed registry: %v",
			err,
		)
	}

	loaded, ok := registry.Get("OBS")
	if !ok || loaded == nil {
		t.Fatal("observed registry token was not loaded")
	}

	if loaded.TotalSupply() != 40 ||
		loaded.BalanceOf(canonicalAddressFixture("owner")) != 40 {
		t.Fatal("observed registry loaded incorrect token state")
	}

	assertRegistryLoadObservationV1(
		t,
		directory,
		TokenRegistryLoadAcceptedEventV1,
		security.SeverityInfo,
		"LOADED",
		"NONE",
		1,
	)
}

func TestRegistryLoadAIObservabilityV1Rejected(
	t *testing.T,
) {
	directory := t.TempDir()

	writeRegistryStateJSONV1(
		t,
		directory,
		map[string]any{
			"BAD": registryStateJSONV1(
				"BAD",
				canonicalAddressFixture("owner"),
				1000,
				40,
				map[string]uint64{
					canonicalAddressFixture("owner"): 39,
				},
				nil,
			),
		},
	)

	registry := registryWithPreservedTokenV1(t)

	err := registry.Load(directory)
	if !errors.Is(
		err,
		ErrRegistryLoadIntegrityV1,
	) {
		t.Fatalf(
			"invalid observed registry error = %v",
			err,
		)
	}

	assertPreservedRegistryV1(
		t,
		registry,
	)

	assertRegistryLoadObservationV1(
		t,
		directory,
		TokenRegistryLoadRejectedEventV1,
		security.SeverityHigh,
		"INTEGRITY_REJECTED",
		"TOKEN_REGISTRY_INTEGRITY",
		1,
	)
}
