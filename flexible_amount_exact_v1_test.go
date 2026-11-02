package main

import (
	"encoding/json"
	"testing"
)

func TestFlexibleAmountPreservesExactJSONNumberV1(t *testing.T) {
	var req SendRequest

	err := json.Unmarshal(
		[]byte(`{"from":"a","to":"b","amount":0.000001}`),
		&req,
	)
	if err != nil {
		t.Fatal(err)
	}

	if req.Amount.String() != "0.000001" {
		t.Fatalf(
			"amount=%q want=%q",
			req.Amount.String(),
			"0.000001",
		)
	}
}

func TestFlexibleAmountPreservesExactJSONStringV1(t *testing.T) {
	var req SendRequest

	err := json.Unmarshal(
		[]byte(`{"from":"a","to":"b","amount":"0.000010"}`),
		&req,
	)
	if err != nil {
		t.Fatal(err)
	}

	if req.Amount.String() != "0.000010" {
		t.Fatalf(
			"amount=%q want=%q",
			req.Amount.String(),
			"0.000010",
		)
	}
}

func TestFlexibleAmountPreservesWholeQCNumberV1(t *testing.T) {
	var req SendRequest

	err := json.Unmarshal(
		[]byte(`{"from":"a","to":"b","amount":1}`),
		&req,
	)
	if err != nil {
		t.Fatal(err)
	}

	if req.Amount.String() != "1" {
		t.Fatalf(
			"amount=%q want=%q",
			req.Amount.String(),
			"1",
		)
	}
}

func TestFlexibleAmountDoesNotFloatRoundV1(t *testing.T) {
	var req SendRequest

	err := json.Unmarshal(
		[]byte(`{"from":"a","to":"b","amount":123.456789}`),
		&req,
	)
	if err != nil {
		t.Fatal(err)
	}

	if req.Amount.String() != "123.456789" {
		t.Fatalf(
			"amount=%q want exact token",
			req.Amount.String(),
		)
	}
}

func TestFlexibleAmountRejectsNonNumberNonStringV1(t *testing.T) {
	for _, body := range []string{
		`{"amount":true}`,
		`{"amount":null}`,
		`{"amount":{}}`,
		`{"amount":[]}`,
	} {
		var req SendRequest

		if err := json.Unmarshal(
			[]byte(body),
			&req,
		); err == nil {
			t.Fatalf(
				"body=%s unexpectedly accepted",
				body,
			)
		}
	}
}

func TestFlexibleAmountRejectsInvalidQCAmountsV1(
	t *testing.T,
) {
	for _, body := range []string{
		`{"amount":-1}`,
		`{"amount":0}`,
		`{"amount":"-1"}`,
		`{"amount":"0"}`,
		`{"amount":"NaN"}`,
		`{"amount":"0.0000001"}`,
		`{"amount":"1e-6"}`,
	} {
		var req SendRequest

		if err := json.Unmarshal(
			[]byte(body),
			&req,
		); err == nil {
			t.Fatalf(
				"body=%s unexpectedly accepted",
				body,
			)
		}
	}
}
