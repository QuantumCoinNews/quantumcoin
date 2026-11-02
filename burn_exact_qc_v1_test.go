package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestBurnRequestAcceptsExactMicroQCStringV1(t *testing.T) {
	var req BurnRequest

	err := json.Unmarshal(
		[]byte(`{"from":"sender","amount":"0.000001"}`),
		&req,
	)
	if err != nil {
		t.Fatalf(
			"burn exact QC string rejected: %v",
			err,
		)
	}

	if got := fmt.Sprint(req.Amount); got != "0.000001" {
		t.Fatalf(
			"burn amount=%q want exact 0.000001 QC",
			got,
		)
	}
}

func TestBurnRequestAcceptsExactMicroQCNumberV1(t *testing.T) {
	var req BurnRequest

	err := json.Unmarshal(
		[]byte(`{"from":"sender","amount":0.000001}`),
		&req,
	)
	if err != nil {
		t.Fatalf(
			"burn exact QC number rejected: %v",
			err,
		)
	}

	if got := fmt.Sprint(req.Amount); got != "0.000001" {
		t.Fatalf(
			"burn amount=%q want exact 0.000001 QC",
			got,
		)
	}
}

func TestBurnRequestCarriesSigningKeyV1(t *testing.T) {
	var req BurnRequest

	err := json.Unmarshal(
		[]byte(
			`{"from":"sender","amount":"0.000001","priv_hex":"deadbeef"}`,
		),
		&req,
	)
	if err != nil {
		t.Fatalf("decode signed burn request: %v", err)
	}

	if req.PrivHex != "deadbeef" {
		t.Fatalf(
			"burn priv_hex=%q want=%q",
			req.PrivHex,
			"deadbeef",
		)
	}
}

func TestBurnHandlerUsesCanonicalSignedBurnBuilderV1(t *testing.T) {
	sourceBytes, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}

	source := string(sourceBytes)

	start := strings.Index(
		source,
		"func handleBurn(",
	)
	if start < 0 {
		t.Fatal("handleBurn not found")
	}

	tail := source[start:]

	end := strings.Index(
		tail,
		"func handleStakeStart(",
	)
	if end < 0 {
		t.Fatal("handleStakeStart boundary not found")
	}

	handler := tail[:end]

	for _, required := range []string{
		"blockchain.NewCanonicalBurnTransactionFromQCAmountV1(",
		"req.From",
		"req.Amount.String()",
		"req.PrivHex",
		"wallet.ImportPrivateKeyHex(",
		"tx.Sign(priv)",
		"tx.Verify()",
		"bc.AddTransaction(tx)",
	} {
		if !strings.Contains(handler, required) {
			t.Fatalf(
				"canonical signed burn handler contract missing %q",
				required,
			)
		}
	}

	for _, forbidden := range []string{
		"blockchain.NewTransactionFromQCAmountV1(",
		"cfg.BurnAddress",
		"burn tx'leri imzasız",
	} {
		if strings.Contains(handler, forbidden) {
			t.Fatalf(
				"legacy burn handler contract still contains %q",
				forbidden,
			)
		}
	}
}
