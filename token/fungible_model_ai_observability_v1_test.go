package token

import (
	"testing"

	"quantumcoin/security"
)

func assertFungiblePersistenceAIEventV1(
	t *testing.T,
	directory string,
	eventType string,
	outcome string,
	errorClass string,
) {
	t.Helper()

	eventLog := security.ReadEvents(directory, 10)

	if len(eventLog.Events) != 1 {
		t.Fatalf(
			"event count = %d, want 1",
			len(eventLog.Events),
		)
	}

	event := eventLog.Events[0]

	if event.Type != eventType {
		t.Fatalf(
			"event type = %q, want %q",
			event.Type,
			eventType,
		)
	}

	if event.Source != "token" ||
		event.Subject != "fungible_persistence" {
		t.Fatalf(
			"unexpected event source/subject: %q/%q",
			event.Source,
			event.Subject,
		)
	}

	if event.Metadata["operation"] !=
		TokenOperationFungiblePersistenceV1 {
		t.Fatalf(
			"operation = %v",
			event.Metadata["operation"],
		)
	}

	if event.Metadata["outcome"] != outcome {
		t.Fatalf(
			"outcome = %v, want %q",
			event.Metadata["outcome"],
			outcome,
		)
	}

	if event.Metadata["errorClass"] != errorClass {
		t.Fatalf(
			"error class = %v, want %q",
			event.Metadata["errorClass"],
			errorClass,
		)
	}

	if event.Metadata["aiReadOnly"] != true ||
		event.Metadata["advisoryOnly"] != true ||
		event.Metadata["authorizesOperation"] != false ||
		event.Metadata["changesTokenState"] != false ||
		event.Metadata["changesRegistry"] != false ||
		event.Metadata["changesChain"] != false ||
		event.Metadata["changesConsensus"] != false {
		t.Fatal("AI persistence event exceeded read-only boundary")
	}

	for _, forbidden := range []string{
		"state",
		"balances",
		"allowances",
		"privateKey",
		"seed",
		"signature",
	} {
		if _, exists := event.Metadata[forbidden]; exists {
			t.Fatalf(
				"event exposes forbidden field %q",
				forbidden,
			)
		}
	}
}

func TestFungiblePersistenceAIObservabilityV1Accepted(
	t *testing.T,
) {
	directory := t.TempDir()
	t.Setenv("QC_NODE_DIR", directory)

	owner := canonicalAddressFixture(
		"owner",
	)

	subject := New(
		"MINE",
		"Mineable Token",
		8,
		1000,
		owner,
	)

	raw, err := MarshalFungibleTokenV1(subject)
	if err != nil {
		t.Fatalf("marshal token: %v", err)
	}

	loaded, err := UnmarshalFungibleTokenV1(raw)
	if err != nil {
		t.Fatalf("unmarshal token: %v", err)
	}

	if loaded == nil ||
		loaded.FungibleModelV1() !=
			FungibleTokenModelControlledEmissionV1 {
		t.Fatal("accepted observation changed decode result")
	}

	assertFungiblePersistenceAIEventV1(
		t,
		directory,
		TokenFungiblePersistenceAcceptedEventV1,
		"LOADED",
		"NONE",
	)
}

func TestFungiblePersistenceAIObservabilityV1Rejected(
	t *testing.T,
) {
	directory := t.TempDir()
	t.Setenv("QC_NODE_DIR", directory)

	raw := []byte(
		`{"version":2,"model":"TOKEN","state":{}}`,
	)

	loaded, err := UnmarshalFungibleTokenV1(raw)

	if err == nil {
		t.Fatal("invalid persistence unexpectedly loaded")
	}

	if loaded != nil {
		t.Fatal("rejected observation returned a token")
	}

	assertFungiblePersistenceAIEventV1(
		t,
		directory,
		TokenFungiblePersistenceRejectedEventV1,
		"REJECTED",
		"FUNGIBLE_PERSISTENCE_VERSION",
	)
}
