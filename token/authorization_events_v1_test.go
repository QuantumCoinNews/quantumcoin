package token

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"quantumcoin/security"
)

func TestObserveTokenAuthorizationV1ProducesReadOnlyEvents(
	t *testing.T,
) {
	directory := t.TempDir()
	t.Setenv("QC_NODE_DIR", directory)

	caller := "test-caller-address"
	domain := "test-token-domain"

	testCases := []struct {
		stage     string
		err       error
		eventType string
		severity  security.EventSeverity
	}{
		{
			stage:     tokenAuthorizationStageAuthorizationV1,
			err:       nil,
			eventType: TokenAuthorizationAcceptedEventV1,
			severity:  security.SeverityInfo,
		},
		{
			stage:     tokenAuthorizationStageAuthorizationV1,
			err:       ErrTokenAuthorizationInvalid,
			eventType: TokenAuthorizationRejectedEventV1,
			severity:  security.SeverityHigh,
		},
		{
			stage:     tokenAuthorizationStageAuthorizationV1,
			err:       ErrTokenAuthorizationReplay,
			eventType: TokenAuthorizationReplayRejectedEventV1,
			severity:  security.SeverityHigh,
		},
		{
			stage:     tokenAuthorizationStageOperationV1,
			err:       errors.New("insufficient token balance"),
			eventType: TokenOperationRejectedEventV1,
			severity:  security.SeverityWarning,
		},
	}

	for index, testCase := range testCases {
		observeTokenAuthorizationV1(
			TokenAuthorizationModelQC20V1,
			TokenOperationTransferV1,
			caller,
			domain,
			uint64(index+1),
			testCase.stage,
			testCase.err,
		)
	}

	eventLog := security.ReadEvents(directory, 20)

	if len(eventLog.Events) != len(testCases) {
		t.Fatalf(
			"event count = %d, want %d",
			len(eventLog.Events),
			len(testCases),
		)
	}

	for index, testCase := range testCases {
		event := eventLog.Events[index]

		if event.Type != testCase.eventType {
			t.Fatalf(
				"event %d type = %q, want %q",
				index,
				event.Type,
				testCase.eventType,
			)
		}

		if event.Severity != testCase.severity {
			t.Fatalf(
				"event %d severity = %q, want %q",
				index,
				event.Severity,
				testCase.severity,
			)
		}

		if event.Source != "token" {
			t.Fatalf(
				"event %d source = %q",
				index,
				event.Source,
			)
		}

		if event.Subject != caller {
			t.Fatalf(
				"event %d subject = %q",
				index,
				event.Subject,
			)
		}

		if event.Metadata["aiReadOnly"] != true {
			t.Fatalf(
				"event %d is not AI read-only",
				index,
			)
		}

		if event.Metadata["advisoryOnly"] != true {
			t.Fatalf(
				"event %d is not advisory-only",
				index,
			)
		}

		if event.Metadata["authorizesOperation"] != false {
			t.Fatalf(
				"event %d authorizes an operation",
				index,
			)
		}

		domainHash, ok :=
			event.Metadata["domainSha256"].(string)

		if !ok || domainHash == "" || domainHash == domain {
			t.Fatalf(
				"event %d has invalid domain hash",
				index,
			)
		}
	}

	summary := security.Summary(directory)

	if summary["totalEvents"] != len(testCases) {
		t.Fatalf(
			"summary total events = %v",
			summary["totalEvents"],
		)
	}

	byType, ok := summary["byType"].(map[string]int)
	if !ok {
		t.Fatalf(
			"summary byType type = %T",
			summary["byType"],
		)
	}

	for _, testCase := range testCases {
		if byType[testCase.eventType] != 1 {
			t.Fatalf(
				"summary event count for %q = %d",
				testCase.eventType,
				byType[testCase.eventType],
			)
		}
	}

	raw, err := os.ReadFile(
		filepath.Join(
			directory,
			"security_events.json",
		),
	)
	if err != nil {
		t.Fatalf("read security events: %v", err)
	}

	lower := strings.ToLower(string(raw))

	for _, forbidden := range []string{
		"private_key",
		"privatekey",
		"priv_hex",
		"public_key",
		"publickey",
		"signature",
	} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf(
				"security event contains forbidden key material field %q",
				forbidden,
			)
		}
	}
}
