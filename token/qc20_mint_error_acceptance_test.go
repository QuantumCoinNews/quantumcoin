package token

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"quantumcoin/security"
)

func TestQC20MintErrorsPropagateOnceWithoutStateChanges(t *testing.T) {
	owner := canonicalAddressFixture("mint-error-owner")
	recipient := canonicalAddressFixture("mint-error-recipient")
	attacker := canonicalAddressFixture("mint-error-attacker")
	cases := []struct {
		name     string
		legacy   bool
		nilToken bool
		owner    string
		caller   string
		to       string
		amount   uint64
		want     error
		wantText string
	}{
		{name: "legacy fixed supply", legacy: true, owner: owner, to: recipient, amount: 1, want: ErrQC20FixedSupplyMintDisabled},
		{name: "legacy ownerless", legacy: true, to: recipient, amount: 1, want: ErrTokenOwnerNotConfigured},
		{name: "legacy invalid owner", legacy: true, owner: "owner", to: recipient, amount: 1, want: ErrInvalidTokenAddress},
		{name: "legacy invalid recipient", legacy: true, owner: owner, to: "recipient", amount: 1, want: ErrInvalidTokenAddress},
		{name: "legacy zero amount", legacy: true, owner: owner, to: recipient, want: ErrTokenAmountZero},
		{name: "legacy nil receiver", legacy: true, nilToken: true, to: recipient, amount: 1, wantText: "QC20 token is nil"},
		{name: "direct fixed supply", owner: owner, caller: owner, to: recipient, amount: 1, want: ErrQC20FixedSupplyMintDisabled},
		{name: "direct non owner", owner: owner, caller: attacker, to: recipient, amount: 1, wantText: "not token owner"},
		{name: "direct invalid caller", owner: owner, caller: "caller", to: recipient, amount: 1, want: ErrInvalidTokenAddress},
		{name: "direct nil receiver", nilToken: true, caller: owner, to: recipient, amount: 1, wantText: "QC20 token is nil"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("QC_NODE_DIR", dir)
			var subject *QC20Token
			if !tc.nilToken {
				subject = NewQC20TokenWithOwner("Fixed Token", "FIX", 100, 8, tc.owner)
			}
			before, err := json.Marshal(subject)
			if err != nil {
				t.Fatal(err)
			}
			var operationErr error
			caller := tc.caller
			if tc.legacy {
				caller = tc.owner
				operationErr = subject.Mint(tc.to, tc.amount)
			} else {
				operationErr = subject.MintByOwner(caller, tc.to, tc.amount)
			}
			if tc.want != nil && !errors.Is(operationErr, tc.want) {
				t.Fatalf("mint error = %v, want %v", operationErr, tc.want)
			}
			if tc.wantText != "" && (operationErr == nil || operationErr.Error() != tc.wantText) {
				t.Fatalf("mint error = %v, want %q", operationErr, tc.wantText)
			}
			after, err := json.Marshal(subject)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Fatal("failed mint changed token state")
			}
			log := security.ReadEvents(dir, 20)
			if log.Error != "" || log.WriteError != "" || len(log.Events) != 1 {
				t.Fatalf("want exactly one error event: %+v", log)
			}
			wantClass := "TOKEN_OPERATION_ERROR"
			if errors.Is(operationErr, ErrQC20FixedSupplyMintDisabled) {
				wantClass = "QC20_FIXED_SUPPLY_MINT_DISABLED"
			}
			requireQC20MintRejectedAIEventV1(t, dir, caller, wantClass)
		})
	}
}

func TestQC20MintErrorSurvivesObservationWriteFailure(t *testing.T) {
	owner := canonicalAddressFixture("mint-error-owner")
	recipient := canonicalAddressFixture("mint-error-recipient")
	for _, failure := range []string{"blocked directory", "corrupt log"} {
		for _, method := range []string{"legacy", "direct"} {
			t.Run(failure+"/"+method, func(t *testing.T) {
				dir := t.TempDir()
				preservedPath := filepath.Join(dir, "security_events.json")
				if failure == "blocked directory" {
					dir = filepath.Join(dir, "node-dir-is-a-file")
					preservedPath = dir
				}
				preserved := []byte("{intentionally damaged log")
				if err := os.WriteFile(preservedPath, preserved, 0600); err != nil {
					t.Fatal(err)
				}
				t.Setenv("QC_NODE_DIR", dir)
				subject := NewQC20TokenWithOwner("Fixed Token", "FIX", 100, 8, owner)
				before, err := json.Marshal(subject)
				if err != nil {
					t.Fatal(err)
				}
				var operationErr error
				if method == "legacy" {
					operationErr = subject.Mint(recipient, 1)
				} else {
					operationErr = subject.MintByOwner(owner, recipient, 1)
				}
				if !errors.Is(operationErr, ErrQC20FixedSupplyMintDisabled) {
					t.Fatalf("log failure replaced mint error: %v", operationErr)
				}
				log := security.ReadEvents(dir, 20)
				if log.WriteError == "" {
					t.Fatal("fault injection did not produce an observable log write failure")
				}
				after, err := json.Marshal(subject)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(before, after) {
					t.Fatal("log failure changed token state")
				}
				contents, err := os.ReadFile(preservedPath)
				if err != nil || !bytes.Equal(contents, preserved) {
					t.Fatal("failed observation destroyed existing file")
				}
			})
		}
	}
}

func TestQC20ConcurrentMintRejectionsReportOneEventPerCall(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("QC_NODE_DIR", dir)
	owner := canonicalAddressFixture("mint-error-owner")
	recipient := canonicalAddressFixture("mint-error-recipient")
	subject := NewQC20TokenWithOwner("Fixed Token", "FIX", 100, 8, owner)
	before, err := json.Marshal(subject)
	if err != nil {
		t.Fatal(err)
	}
	const calls = 16
	var wg sync.WaitGroup
	for range calls {
		wg.Go(func() {
			if err := subject.Mint(recipient, 1); !errors.Is(err, ErrQC20FixedSupplyMintDisabled) {
				t.Errorf("concurrent mint error: %v", err)
			}
		})
	}
	wg.Wait()
	log := security.ReadEvents(dir, 100)
	if log.Error != "" || log.WriteError != "" || len(log.Events) != calls {
		t.Fatalf("event count = %d, want %d; read=%s write=%s", len(log.Events), calls, log.Error, log.WriteError)
	}
	seen := make(map[string]bool)
	for _, ev := range log.Events {
		if ev.ID == "" || seen[ev.ID] || ev.Type != TokenOperationRejectedEventV1 || ev.Metadata["errorClass"] != "QC20_FIXED_SUPPLY_MINT_DISABLED" {
			t.Fatal("invalid, duplicate or misleading rejection event")
		}
		seen[ev.ID] = true
	}
	after, err := json.Marshal(subject)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("concurrent mint rejection changed state")
	}
}
