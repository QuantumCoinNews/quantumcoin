package main

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"quantumcoin/blockchain"
	qnode "quantumcoin/node"
)

func queryResourceMuxV1(t *testing.T, chain *blockchain.Blockchain) http.Handler {
	t.Helper()
	clearActualMuxTestEnvV1(t)
	t.Setenv("QC_API_TOKEN", actualMuxTestTokenV1)
	previous, readOnly := nodeRuntime, apiReadOnlyMode
	nodeRuntime, apiReadOnlyMode = qnode.NewRuntime(chain, nil), false
	t.Cleanup(func() { nodeRuntime, apiReadOnlyMode = previous, readOnly })
	return newActualMuxTestHandlerV1(t)
}

func queryResourceRequestV1(handler http.Handler, method, target string) *httptest.ResponseRecorder {
	request := newActualMuxRequestV1(method, target, nil)
	addActualMuxTokenV1(request)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestQueryResourceBlocksRealMuxBoundedPages(t *testing.T) {
	// Projection-only fixture; no blocks/transactions are admitted to consensus.
	// Large transaction lists must not become part of a summary response.
	chain := &blockchain.Blockchain{}
	txs := make([]*blockchain.Transaction, 8192)
	for i := range 257 {
		chain.Blocks = append(chain.Blocks, &blockchain.Block{Index: i, Hash: make([]byte, 32), PrevHash: make([]byte, 32), Miner: "summary-fixture", Transactions: txs})
	}
	handler := queryResourceMuxV1(t, chain)
	for _, test := range []struct {
		name, query                       string
		limit, offset, count, first, last int
		hasMore                           bool
	}{
		{"default", "", 20, 0, 20, 237, 256, true},
		{"single", "?limit=1", 1, 0, 1, 256, 256, true},
		{"next-page", "?limit=2&offset=1", 2, 1, 2, 254, 255, true},
		{"cap", "?limit=999999999", 200, 0, 200, 57, 256, true},
		{"invalid", "?limit=invalid&offset=-1", 20, 0, 20, 237, 256, true},
		{"overflow", "?limit=" + strings.Repeat("9", 80), 20, 0, 20, 237, 256, true},
		{"tail", "?limit=200&offset=256", 200, 256, 1, 0, 0, false},
		{"past-end", "?offset=257", 20, 257, 0, 0, 0, false},
		{"max-offset", "?offset=" + strconv.Itoa(int(^uint(0)>>1)), 20, int(^uint(0) >> 1), 0, 0, 0, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := queryResourceRequestV1(handler, http.MethodGet, "/api/blocks"+test.query)
			if response.Code != http.StatusOK {
				t.Fatalf("status %d: %s", response.Code, response.Body.String())
			}
			var body []struct {
				Index   int `json:"index"`
				TxCount int `json:"tx_count"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body == nil || len(body) != test.count {
				t.Fatalf("unexpected page: %+v", body)
			}
			if len(body) > 0 && (body[0].Index != test.first || body[len(body)-1].Index != test.last) {
				t.Fatal("wrong page order/boundaries")
			}
			for _, item := range body {
				if item.TxCount != len(txs) {
					t.Fatal("lost transaction cardinality")
				}
			}
			for header, want := range map[string]string{"X-Total-Count": "257", "X-Page-Limit": strconv.Itoa(test.limit), "X-Page-Offset": strconv.Itoa(test.offset), "X-Has-More": strconv.FormatBool(test.hasMore)} {
				if got := response.Header().Get(header); got != want {
					t.Fatalf("%s=%s want=%s", header, got, want)
				}
			}
			wantNext := ""
			if test.hasMore {
				wantNext = strconv.Itoa(test.offset + test.count)
			}
			if response.Header().Get("X-Next-Offset") != wantNext {
				t.Fatal("invalid next-page cursor")
			}
			if response.Body.Len() > 128<<10 || strings.Contains(response.Body.String(), `"transactions"`) {
				t.Fatal("summary leaked transaction payloads")
			}
		})
	}
}

func TestQueryResourceStatusAndRetiredMempoolRealMux(t *testing.T) {
	id := strings.Repeat("12", 32)
	rawID, err := hex.DecodeString(id)
	if err != nil {
		t.Fatal(err)
	}
	chain := &blockchain.Blockchain{Blocks: []*blockchain.Block{{Index: 0, Transactions: []*blockchain.Transaction{{ID: rawID}}}}}
	handler := queryResourceMuxV1(t, chain)
	for _, test := range []struct {
		name, target, status string
		code                 int
	}{
		{"confirmed", "/api/tx/status?id=" + id, "confirmed", 200},
		{"alias", "/api/tx/status?txid=" + id, "confirmed", 200},
		{"unknown", "/api/tx/status?id=" + strings.Repeat("ab", 32), "unknown", 200},
		{"missing", "/api/tx/status", "", 400},
		{"retired-pool", "/api/mempool?limit=999999999", "", 404},
		{"retired-pool-child", "/api/mempool/transactions", "", 404},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := queryResourceRequestV1(handler, http.MethodGet, test.target)
			if response.Code != test.code || !strings.HasPrefix(response.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			var body struct {
				Status          string `json:"status"`
				InBlock         bool   `json:"inBlock"`
				ConfirmedHeight int    `json:"confirmedHeight"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Status != test.status {
				t.Fatalf("status=%s want=%s", body.Status, test.status)
			}
			if test.status == "confirmed" && (!body.InBlock || body.ConfirmedHeight != 0) {
				t.Fatal("incorrect indexed location")
			}
			if response.Body.Len() > 2048 {
				t.Fatal("status query leaked transaction/chain payload")
			}
		})
	}
}
