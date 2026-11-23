package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"quantumcoin/blockchain"
	qnode "quantumcoin/node"
)

func TestMuxSecurityCurrentSignedAmounts(t *testing.T) {
	clearActualMuxTestEnvV1(t)
	chain, owner, to, c := networkSigningFixture(t)
	previous, previousConfig, mode := nodeRuntime, cfg, apiReadOnlyMode
	nodeRuntime, cfg, apiReadOnlyMode = qnode.NewRuntime(chain, c), c, false
	t.Cleanup(func() { nodeRuntime, cfg, apiReadOnlyMode = previous, previousConfig, mode })
	t.Setenv("QC_API_TOKEN", actualMuxTestTokenV1)
	tx, err := blockchain.NewTransactionFromQCAmountV1(owner.GetAddress(), to.GetAddress(), "1", chain)
	if err != nil {
		t.Fatal(err)
	}
	if err = chain.SignTransaction(tx, owner.PrivateKey); err != nil {
		t.Fatal(err)
	}
	if !chain.VerifyTransaction(tx) {
		t.Fatal("positive fixture signature invalid")
	}
	envelope := signedEnvelope(t, tx)
	if err = chain.SaveToFile(c.ChainFile); err != nil {
		t.Fatal(err)
	}
	disk, err := os.ReadFile(c.ChainFile)
	if err != nil {
		t.Fatal(err)
	}
	before := chain.StateSnapshot()
	var negative SendRequest
	if err = json.Unmarshal([]byte(envelope), &negative); err != nil {
		t.Fatal(err)
	}
	negative.Transaction.Outputs[0].Amount = -1
	negative.Transaction.ID = negative.Transaction.Hash()
	if err = chain.SignTransaction(negative.Transaction, owner.PrivateKey); err != nil {
		t.Fatal(err)
	}
	if !chain.VerifyTransaction(negative.Transaction) {
		t.Fatal("negative output fixture must have a valid owner signature")
	}
	numericBody := func(raw string) string {
		t.Helper()
		var outer map[string]json.RawMessage
		var fields map[string]json.RawMessage
		if err := json.Unmarshal([]byte(envelope), &outer); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(outer["transaction"], &fields); err != nil {
			t.Fatal(err)
		}
		fields["Amount"] = json.RawMessage(`"__MUX_NUMERIC__"`)
		encoded, err := json.Marshal(fields)
		if err != nil {
			t.Fatal(err)
		}
		outer["transaction"] = encoded
		encoded, err = json.Marshal(outer)
		if err != nil {
			t.Fatal(err)
		}
		return strings.Replace(string(encoded), `"__MUX_NUMERIC__"`, raw, 1)
	}
	for _, row := range []struct{ name, body string }{
		{"owner-signed-negative-output", signedEnvelope(t, negative.Transaction)},
		{"nested-raw-NaN", numericBody("NaN")}, {"nested-string-NaN", numericBody(`"NaN"`)}, {"nested-overflow", numericBody("1e999")},
	} {
		t.Run(row.name, func(t *testing.T) {
			h := newHTTPAPIHandlerV1(NewHTTPGuard(), NewEndpointLimiter())
			response := webRequest(h, "POST", "/api/tx/send", row.body)
			assertRoutingJSONErrorV1(t, response.Code, response.Header(), response.Body.Bytes(), 400)
			after, err := os.ReadFile(c.ChainFile)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, chain.StateSnapshot()) || !bytes.Equal(disk, after) {
				t.Fatal("invalid signed-schema numeric input changed chain/mempool/file")
			}
		})
	}
	response := webRequest(newHTTPAPIHandlerV1(NewHTTPGuard(), NewEndpointLimiter()), "POST", "/api/tx/send", envelope)
	if response.Code != 200 || chain.PendingTransactionCountSnapshot() != 1 {
		t.Fatal("valid owner-signed positive control failed", response.Code, response.Body.String())
	}
}

func TestMuxSecuritySignedPipelineOverTCP(t *testing.T) {
	_, chain, envelope, file := corsSignedRequestFixtureV1(t)
	disk, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	before := chain.StateSnapshot()
	oversized := envelope + strings.Repeat(" ", int(apiMaxBodyBytes)+1-len(envelope))
	for _, row := range []struct {
		name, method, token, origin, body string
		readonly, chunked                 bool
		status                            int
	}{
		{"auth", "POST", "wrong", corsAcceptanceOriginV1, envelope, false, false, 401},
		{"cors", "POST", actualMuxTestTokenV1, "https://untrusted.example", envelope, false, false, 403},
		{"method", "GET", actualMuxTestTokenV1, corsAcceptanceOriginV1, envelope, false, false, 405},
		{"known-length", "POST", actualMuxTestTokenV1, corsAcceptanceOriginV1, oversized, false, false, 413},
		{"chunked", "POST", actualMuxTestTokenV1, corsAcceptanceOriginV1, oversized, false, true, 413},
		{"read-only", "POST", actualMuxTestTokenV1, corsAcceptanceOriginV1, envelope, true, false, 403},
	} {
		t.Run(row.name, func(t *testing.T) {
			nodeRuntime.SetAPIReadOnly(row.readonly)
			defer nodeRuntime.SetAPIReadOnly(false)
			h := newHTTPAPIHandlerForListenV1(NewHTTPGuard(), NewEndpointLimiter(), "0.0.0.0:8082")
			server := httptest.NewServer(h)
			defer server.Close()
			req, err := http.NewRequest(row.method, server.URL+"/api/tx/send", strings.NewReader(row.body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("X-QC-API-KEY", row.token)
			req.Header.Set("Origin", row.origin)
			req.Header.Set("Content-Type", "application/json")
			if row.chunked {
				req.ContentLength = -1
			}
			client := &http.Client{Timeout: 5 * time.Second}
			response, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			assertRoutingJSONErrorV1(t, response.StatusCode, response.Header, body, row.status)
			after, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, chain.StateSnapshot()) || !bytes.Equal(disk, after) {
				t.Fatal("denied pipeline changed ledger")
			}
		})
	}
}

func TestMuxSecurityConcurrentSignedAdmissionsOverTCP(t *testing.T) {
	_, chain, envelope, file := corsSignedRequestFixtureV1(t)
	disk, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	blocks := chain.AllBlocksSnapshot()
	var signed SendRequest
	if err = json.Unmarshal([]byte(envelope), &signed); err != nil {
		t.Fatal(err)
	}
	expectedID := hex.EncodeToString(signed.Transaction.ID)
	server := httptest.NewServer(newHTTPAPIHandlerForListenV1(NewHTTPGuard(), NewEndpointLimiter(), "0.0.0.0:8082"))
	defer server.Close()
	type result struct {
		kind   string
		status int
		header http.Header
		body   []byte
		err    error
	}
	results := make(chan result, 12)
	start := make(chan struct{})
	var workers sync.WaitGroup
	client := &http.Client{Timeout: 10 * time.Second}
	for _, kind := range []string{"valid", "bad-auth", "bad-origin"} {
		for range 4 {
			workers.Go(func() {
				<-start
				row := result{kind: kind}
				req, err := http.NewRequest("POST", server.URL+"/api/tx/send", strings.NewReader(envelope))
				if err != nil {
					row.err = err
					results <- row
					return
				}
				req.Header.Set("X-QC-API-KEY", actualMuxTestTokenV1)
				req.Header.Set("Origin", corsAcceptanceOriginV1)
				req.Header.Set("Content-Type", "application/json")
				if kind == "bad-auth" {
					req.Header.Set("X-QC-API-KEY", "wrong")
				}
				if kind == "bad-origin" {
					req.Header.Set("Origin", "https://untrusted.example")
				}
				response, err := client.Do(req)
				if err != nil {
					row.err = err
					results <- row
					return
				}
				row.status, row.header = response.StatusCode, response.Header
				row.body, row.err = io.ReadAll(response.Body)
				_ = response.Body.Close()
				results <- row
			})
		}
	}
	close(start)
	workers.Wait()
	close(results)
	accepted := 0
	for row := range results {
		if row.err != nil {
			t.Error(row.err)
			continue
		}
		if row.kind == "bad-origin" && row.status != 403 {
			t.Errorf("origin bypass: %d", row.status)
		}
		if row.kind == "bad-auth" && row.status != 401 && row.status != 429 {
			t.Errorf("auth bypass: %d", row.status)
		}
		if row.kind == "valid" && row.status != 200 && row.status != 400 && row.status != 429 {
			t.Errorf("unexpected valid/replay/rate status: %d", row.status)
		}
		if row.status == 200 {
			var receipt SendResponse
			if row.kind != "valid" || json.Unmarshal(row.body, &receipt) != nil || !receipt.Success || receipt.TxID != expectedID {
				t.Errorf("invalid acceptance receipt: %s", row.body)
			}
			accepted++
		} else {
			assertRoutingJSONErrorV1(t, row.status, row.header, row.body, row.status)
		}
	}
	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if accepted < 1 || chain.PendingTransactionCountSnapshot() != 1 || !reflect.DeepEqual(blocks, chain.AllBlocksSnapshot()) || !bytes.Equal(disk, after) {
		t.Fatalf("concurrent admission drift: accepted=%d pending=%d", accepted, chain.PendingTransactionCountSnapshot())
	}
}
