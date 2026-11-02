package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"quantumcoin/blockchain"
	qnode "quantumcoin/node"
)

const actualMuxTestTokenV1 = "qma0038-actual-mux-token"

func clearActualMuxTestEnvV1(t *testing.T) {
	t.Helper()

	keys := []string{
		"QC_API_TOKEN",
		"QC_DEV_MODE",
		"QC_PUBLIC_TESTNET",
		"QC_API_PUBLIC_TESTNET",
		"QC_TESTNET_PUBLIC_API",
		"QC_TESTNET",
		"QC_NET_MAGIC",
		"QC_NETWORK",
		"QC_NETWORK_MODE",
		"QC_CHAIN_MODE",
		"QC_MODE",
		"QC_API_LISTEN",
		"QC_CORS_ALLOWED_ORIGINS",
		"QC_CORS_ALLOWED_ORIGIN",
	}

	old := make(map[string]string, len(keys))
	had := make(map[string]bool, len(keys))

	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			old[key] = value
			had[key] = true
		}
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	}

	t.Cleanup(func() {
		for _, key := range keys {
			if had[key] {
				_ = os.Setenv(key, old[key])
			} else {
				_ = os.Unsetenv(key)
			}
		}
	})
}

func newActualMuxTestHandlerV1(t *testing.T) http.Handler {
	t.Helper()

	return newHTTPAPIHandlerV1(
		NewHTTPGuard(),
		NewEndpointLimiter(),
	)
}

func newActualMuxRequestV1(
	method string,
	target string,
	body *bytes.Reader,
) *http.Request {
	var request *http.Request

	if body == nil {
		request = httptest.NewRequest(method, target, nil)
	} else {
		request = httptest.NewRequest(method, target, body)
	}

	request.RemoteAddr = "127.0.0.1:42038"
	request.Host = "127.0.0.1:8080"

	return request
}

func addActualMuxTokenV1(request *http.Request) {
	request.Header.Set(
		"X-QC-API-KEY",
		actualMuxTestTokenV1,
	)
}

func TestActualMuxAuthIntegrationV1(t *testing.T) {
	clearActualMuxTestEnvV1(t)

	handler := newActualMuxTestHandlerV1(t)

	missingRequest := newActualMuxRequestV1(
		http.MethodGet,
		"/api/economy/policy",
		nil,
	)
	missingRecorder := httptest.NewRecorder()

	handler.ServeHTTP(missingRecorder, missingRequest)

	if missingRecorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"missing token status=%d want=%d body=%q",
			missingRecorder.Code,
			http.StatusUnauthorized,
			missingRecorder.Body.String(),
		)
	}

	t.Setenv("QC_API_TOKEN", actualMuxTestTokenV1)

	validRequest := newActualMuxRequestV1(
		http.MethodGet,
		"/api/economy/policy",
		nil,
	)
	addActualMuxTokenV1(validRequest)
	validRecorder := httptest.NewRecorder()

	handler.ServeHTTP(validRecorder, validRequest)

	if validRecorder.Code != http.StatusOK {
		t.Fatalf(
			"valid token status=%d want=%d body=%q",
			validRecorder.Code,
			http.StatusOK,
			validRecorder.Body.String(),
		)
	}

	bearerRequest := newActualMuxRequestV1(
		http.MethodGet,
		"/api/economy/policy",
		nil,
	)
	bearerRequest.Header.Set(
		"Authorization",
		"Bearer "+actualMuxTestTokenV1,
	)
	bearerRecorder := httptest.NewRecorder()

	handler.ServeHTTP(bearerRecorder, bearerRequest)

	if bearerRecorder.Code != http.StatusOK {
		t.Fatalf(
			"bearer token status=%d want=%d body=%q",
			bearerRecorder.Code,
			http.StatusOK,
			bearerRecorder.Body.String(),
		)
	}
}

func TestActualMuxOversizedBodyReturns413V1(t *testing.T) {
	clearActualMuxTestEnvV1(t)
	t.Setenv("QC_API_TOKEN", actualMuxTestTokenV1)

	oversized := bytes.Repeat(
		[]byte("x"),
		int(apiMaxBodyBytes)+1,
	)

	tests := []struct {
		name          string
		contentLength int64
	}{
		{
			name:          "known_length",
			contentLength: int64(len(oversized)),
		},
		{
			name:          "unknown_length",
			contentLength: -1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := newActualMuxTestHandlerV1(t)
			request := newActualMuxRequestV1(
				http.MethodPost,
				"/api/tx/send",
				bytes.NewReader(oversized),
			)
			request.ContentLength = test.contentLength
			request.Header.Set(
				"Content-Type",
				"application/json",
			)
			addActualMuxTokenV1(request)

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf(
					"status=%d want=%d body=%q",
					recorder.Code,
					http.StatusRequestEntityTooLarge,
					recorder.Body.String(),
				)
			}
		})
	}
}

func TestActualMuxNonPreflightCORSV1(t *testing.T) {
	clearActualMuxTestEnvV1(t)

	t.Setenv("QC_PUBLIC_TESTNET", "1")
	t.Setenv(
		"QC_CORS_ALLOWED_ORIGINS",
		"https://wallet.quantumcoin.test",
	)

	handler := newActualMuxTestHandlerV1(t)

	rejectedRequest := newActualMuxRequestV1(
		http.MethodGet,
		"/api/does-not-exist",
		nil,
	)
	rejectedRequest.Header.Set(
		"Origin",
		"https://evil.example",
	)
	rejectedRecorder := httptest.NewRecorder()

	handler.ServeHTTP(
		rejectedRecorder,
		rejectedRequest,
	)

	if rejectedRecorder.Code != http.StatusForbidden {
		t.Fatalf(
			"disallowed origin status=%d want=%d body=%q",
			rejectedRecorder.Code,
			http.StatusForbidden,
			rejectedRecorder.Body.String(),
		)
	}

	allowedRequest := newActualMuxRequestV1(
		http.MethodGet,
		"/api/does-not-exist",
		nil,
	)
	allowedRequest.Header.Set(
		"Origin",
		"https://wallet.quantumcoin.test",
	)
	allowedRecorder := httptest.NewRecorder()

	handler.ServeHTTP(
		allowedRecorder,
		allowedRequest,
	)

	if allowedRecorder.Code != http.StatusNotFound {
		t.Fatalf(
			"allowed origin status=%d want=%d body=%q",
			allowedRecorder.Code,
			http.StatusNotFound,
			allowedRecorder.Body.String(),
		)
	}

	if got := allowedRecorder.Header().Get(
		"Access-Control-Allow-Origin",
	); got != "https://wallet.quantumcoin.test" {
		t.Fatalf(
			"allow-origin=%q",
			got,
		)
	}
}

func TestActualMuxSendAndMinerMethodContractsV1(t *testing.T) {
	clearActualMuxTestEnvV1(t)

	t.Setenv("QC_API_TOKEN", actualMuxTestTokenV1)
	t.Setenv("QC_DEV_MODE", "1")

	tests := []struct {
		name      string
		method    string
		path      string
		wantAllow string
	}{
		{
			name:   "send_requires_post",
			method: http.MethodGet,
			path:   "/api/tx/send",
		},
		{
			name:      "miner_start_requires_post",
			method:    http.MethodGet,
			path:      "/api/miner/start",
			wantAllow: http.MethodPost,
		},
		{
			name:      "miner_stop_requires_post",
			method:    http.MethodGet,
			path:      "/api/miner/stop",
			wantAllow: http.MethodPost,
		},
		{
			name:      "miner_status_requires_get",
			method:    http.MethodPost,
			path:      "/api/miner/status",
			wantAllow: http.MethodGet,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := newActualMuxTestHandlerV1(t)
			request := newActualMuxRequestV1(
				test.method,
				test.path,
				nil,
			)
			addActualMuxTokenV1(request)

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusMethodNotAllowed {
				t.Fatalf(
					"status=%d want=%d body=%q",
					recorder.Code,
					http.StatusMethodNotAllowed,
					recorder.Body.String(),
				)
			}

			if test.wantAllow != "" {
				if got := recorder.Header().Get("Allow"); got != test.wantAllow {
					t.Fatalf(
						"Allow=%q want=%q",
						got,
						test.wantAllow,
					)
				}
			}
		})
	}
}

func TestActualMuxRejectsNegativeAndNaNSendAmountsV1(t *testing.T) {
	clearActualMuxTestEnvV1(t)
	t.Setenv("QC_API_TOKEN", actualMuxTestTokenV1)

	tests := []struct {
		name string
		body string
	}{
		{
			name: "negative",
			body: `{"from":"from","to":"to","amount":-1,"priv_hex":"00"}`,
		},
		{
			name: "raw_nan",
			body: `{"from":"from","to":"to","amount":NaN,"priv_hex":"00"}`,
		},
		{
			name: "string_nan",
			body: `{"from":"from","to":"to","amount":"NaN","priv_hex":"00"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := newActualMuxTestHandlerV1(t)
			request := newActualMuxRequestV1(
				http.MethodPost,
				"/api/tx/send",
				bytes.NewReader([]byte(test.body)),
			)
			request.Header.Set(
				"Content-Type",
				"application/json",
			)
			addActualMuxTokenV1(request)

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf(
					"status=%d want=%d body=%q",
					recorder.Code,
					http.StatusBadRequest,
					recorder.Body.String(),
				)
			}
		})
	}
}

func TestActualMuxUnknownAPINeverUsesSPAV1(t *testing.T) {
	clearActualMuxTestEnvV1(t)

	handler := newActualMuxTestHandlerV1(t)
	request := newActualMuxRequestV1(
		http.MethodGet,
		"/api/unknown/nested",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf(
			"status=%d want=%d body=%q",
			recorder.Code,
			http.StatusNotFound,
			recorder.Body.String(),
		)
	}

	contentType := recorder.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Fatalf(
			"Content-Type=%q want JSON",
			contentType,
		)
	}

	if strings.Contains(
		strings.ToLower(recorder.Body.String()),
		"<html",
	) {
		t.Fatalf(
			"unknown API returned SPA HTML: %q",
			recorder.Body.String(),
		)
	}
}

func TestActualMuxSPAFallbackV1(t *testing.T) {
	clearActualMuxTestEnvV1(t)

	handler := newActualMuxTestHandlerV1(t)
	request := newActualMuxRequestV1(
		http.MethodGet,
		"/wallet/receive",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status=%d want=%d body=%q",
			recorder.Code,
			http.StatusOK,
			recorder.Body.String(),
		)
	}

	contentType := recorder.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Fatalf(
			"Content-Type=%q want HTML",
			contentType,
		)
	}

	body := strings.ToLower(recorder.Body.String())
	if !strings.Contains(body, "<html") &&
		!strings.Contains(body, "<!doctype") {
		t.Fatalf(
			"SPA fallback did not return index HTML: %q",
			recorder.Body.String(),
		)
	}
}

func TestActualMuxPaginationHeadersV1(t *testing.T) {
	clearActualMuxTestEnvV1(t)
	t.Setenv("QC_API_TOKEN", actualMuxTestTokenV1)

	originalBlockchain := bc
	bc = &blockchain.Blockchain{}
	t.Cleanup(func() {
		bc = originalBlockchain
	})

	handler := newActualMuxTestHandlerV1(t)
	request := newActualMuxRequestV1(
		http.MethodGet,
		"/api/blocks?limit=500000&offset=-2",
		nil,
	)
	addActualMuxTokenV1(request)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status=%d want=%d body=%q",
			recorder.Code,
			http.StatusOK,
			recorder.Body.String(),
		)
	}

	expectedHeaders := map[string]string{
		"X-Total-Count": "0",
		"X-Page-Limit":  "200",
		"X-Page-Offset": "0",
		"X-Has-More":    "false",
	}

	for name, want := range expectedHeaders {
		if got := recorder.Header().Get(name); got != want {
			t.Fatalf(
				"%s=%q want=%q",
				name,
				got,
				want,
			)
		}
	}
}

func TestProductionHTTPTimeoutOptionsV1(t *testing.T) {
	options := qnode.DefaultHTTPServerOptionsV1()

	if options.ReadHeaderTimeout <= 0 ||
		options.ReadTimeout <= 0 ||
		options.WriteTimeout <= 0 ||
		options.IdleTimeout <= 0 {
		t.Fatalf(
			"production HTTP timeouts are not explicit: %+v",
			options,
		)
	}
}

func TestActualMuxConcurrentRequestsV1(t *testing.T) {
	clearActualMuxTestEnvV1(t)
	t.Setenv("QC_API_TOKEN", actualMuxTestTokenV1)

	handler := newActualMuxTestHandlerV1(t)

	const (
		workers  = 12
		requests = 4
	)

	var wait sync.WaitGroup
	errorsChannel := make(chan error, workers*requests)

	for worker := 0; worker < workers; worker++ {
		worker := worker

		wait.Add(1)

		go func() {
			defer wait.Done()

			for requestIndex := 0; requestIndex < requests; requestIndex++ {
				request := newActualMuxRequestV1(
					http.MethodGet,
					"/api/economy/policy",
					nil,
				)
				request.RemoteAddr = fmt.Sprintf(
					"127.0.0.%d:%d",
					(worker%200)+1,
					43000+requestIndex,
				)
				addActualMuxTokenV1(request)

				recorder := httptest.NewRecorder()
				handler.ServeHTTP(recorder, request)

				if recorder.Code != http.StatusOK {
					errorsChannel <- fmt.Errorf(
						"worker=%d request=%d status=%d body=%q",
						worker,
						requestIndex,
						recorder.Code,
						recorder.Body.String(),
					)
				}
			}
		}()
	}

	wait.Wait()
	close(errorsChannel)

	for err := range errorsChannel {
		t.Error(err)
	}
}

func TestActualMuxBurnCriticalRateLimitV1(t *testing.T) {
	clearActualMuxTestEnvV1(t)
	t.Setenv("QC_API_TOKEN", actualMuxTestTokenV1)

	handler := newActualMuxTestHandlerV1(t)

	for requestNumber := 1; requestNumber <= 6; requestNumber++ {
		request := newActualMuxRequestV1(
			http.MethodPost,
			"/api/tx/burn",
			bytes.NewReader([]byte(`{}`)),
		)
		addActualMuxTokenV1(request)

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		if requestNumber <= 5 {
			if recorder.Code == http.StatusTooManyRequests {
				t.Fatalf(
					"burn request %d was rate limited before configured maximum",
					requestNumber,
				)
			}
			continue
		}

		if recorder.Code != http.StatusTooManyRequests {
			t.Fatalf(
				"burn request %d status=%d want=%d body=%q",
				requestNumber,
				recorder.Code,
				http.StatusTooManyRequests,
				recorder.Body.String(),
			)
		}
	}
}
