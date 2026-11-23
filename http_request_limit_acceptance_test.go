package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	qnode "quantumcoin/node"
)

func TestSmallJSONBodyLimit(t *testing.T) {
	externalMiningMux(t)
	t.Setenv("QC_DIFFICULTY_BITS", "")
	runtime, err := qnode.BootstrapWithConfig(cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	nodeRuntime = runtime
	t.Setenv("QC_MONITORING_ADMIN_TOKEN", "request-limit-operator")
	t.Setenv("QC_PUBLISHER_MONITOR_TOKEN", "request-limit-publisher")
	for _, path := range []string{"/api/monitoring/action", "/api/monitoring/publisher"} {
		t.Run(path, func(t *testing.T) {
			valid := `{"id":"absent","action":"ack","note":"checked"}`
			acceptedStatus := 409 // Parsed action, but no such incident.
			if strings.HasSuffix(path, "publisher") {
				valid = `{"version":"QC_PUBLISHER_REPORT_V1","jobId":"2026-11-16.slot1","editorial":"passed","deliveries":[{"platform":"x","status":"uncertain"}]}`
				acceptedStatus = 200
			}
			for _, row := range []struct {
				name, body string
				status     int
			}{
				{"oversized-value", `{"note":"` + strings.Repeat("x", 4097) + `"}`, 413},
				{"oversized-prefix", strings.Repeat(" ", 4097) + valid, 413},
				{"oversized-suffix", valid + strings.Repeat(" ", 4097), 413},
				{"malformed", `{`, 400},
				{"trailing-object", valid + `{}`, 400},
				{"exact-limit", valid + strings.Repeat(" ", 4096-len(valid)), acceptedStatus},
			} {
				t.Run(row.name, func(t *testing.T) {
					h := newHTTPAPIHandlerV1(NewHTTPGuard(), NewEndpointLimiter())
					before := runtime.Monitoring().Snapshot()
					req := httptest.NewRequest("POST", "http://127.0.0.1"+path, strings.NewReader(row.body))
					req.RemoteAddr = "127.0.0.1:34444"
					req.ContentLength = -1
					req.Header.Set("X-QC-API-KEY", actualMuxTestTokenV1)
					req.Header.Set("X-QC-MONITORING-ADMIN", "request-limit-operator")
					req.Header.Set("X-QC-PUBLISHER-MONITOR", "request-limit-publisher")
					w := httptest.NewRecorder()
					h.ServeHTTP(w, req)
					if w.Code != row.status {
						t.Fatalf("status=%d want=%d: %s", w.Code, row.status, w.Body.String())
					}
					if row.status != 200 && !reflect.DeepEqual(before, runtime.Monitoring().Snapshot()) {
						t.Fatal("rejected body changed monitoring state")
					}
				})
			}
		})
	}
}

func TestHTTPRequestBodyLimitOverTCP(t *testing.T) {
	for _, chunked := range []bool{false, true} {
		for _, size := range []int{int(httpMaxBodyBytes), int(httpMaxBodyBytes) + 1} {
			name := "known"
			if chunked {
				name = "chunked"
			}
			if size > int(httpMaxBodyBytes) {
				name += "-over"
			} else {
				name += "-exact"
			}
			t.Run(name, func(t *testing.T) {
				var committed atomic.Bool
				handler := NewHTTPGuard().Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					data, err := io.ReadAll(r.Body)
					if err != nil {
						if !writeRequestBodyTooLargeV1(w, err) {
							t.Error(err)
							writeError(w, 400, "read error")
						}
						return
					}
					if len(data) != size {
						t.Errorf("read=%d want=%d", len(data), size)
					}
					committed.Store(true)
					w.WriteHeader(204)
				}))
				server := httptest.NewServer(withEarlyRejectionBodyDrainV1(handler))
				defer server.Close()
				request, err := http.NewRequest("POST", server.URL, strings.NewReader(strings.Repeat("x", size)))
				if err != nil {
					t.Fatal(err)
				}
				if chunked {
					request.ContentLength = -1
				}
				client := &http.Client{Timeout: 5 * time.Second}
				response, err := client.Do(request)
				if err != nil {
					t.Fatal(err)
				}
				defer response.Body.Close()
				body, err := io.ReadAll(response.Body)
				if err != nil {
					t.Fatal(err)
				}
				want := 204
				if size > int(httpMaxBodyBytes) {
					want = 413
					var failure struct {
						Success bool
						Message string
					}
					if json.Unmarshal(body, &failure) != nil || failure.Success || failure.Message == "" {
						t.Fatalf("invalid 413 JSON: %s", body)
					}
				}
				if response.StatusCode != want || committed.Load() != (want == 204) {
					t.Fatalf("status=%d want=%d committed=%v", response.StatusCode, want, committed.Load())
				}
			})
		}
	}
}

func TestHTTPRequestLimitSignedTransferPreservesLedger(t *testing.T) {
	h, chain, envelope, path := corsSignedRequestFixtureV1(t)
	beforeFile, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	before := chain.PendingTransactionsSnapshot()
	for _, unknown := range []bool{false, true} {
		req := httptest.NewRequest("POST", "/api/tx/send", strings.NewReader(envelope+strings.Repeat(" ", int(apiMaxBodyBytes)+1)))
		req.RemoteAddr = "127.0.0.1:34444"
		req.Header.Set("X-QC-API-KEY", actualMuxTestTokenV1)
		if unknown {
			req.ContentLength = -1
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != 413 {
			t.Fatalf("unknown=%v status=%d", unknown, w.Code)
		}
		afterFile, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(beforeFile, afterFile) || !reflect.DeepEqual(before, chain.PendingTransactionsSnapshot()) {
			t.Fatal("oversized signed body changed ledger")
		}
	}
	req := httptest.NewRequest("POST", "/api/tx/send", strings.NewReader(envelope+strings.Repeat(" ", int(apiMaxBodyBytes)-len(envelope))))
	req.RemoteAddr = "127.0.0.1:34444"
	req.Header.Set("X-QC-API-KEY", actualMuxTestTokenV1)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 || len(chain.PendingTransactionsSnapshot()) != len(before)+1 {
		t.Fatalf("valid exact-limit transfer not admitted: %d %s", w.Code, w.Body.String())
	}
}
