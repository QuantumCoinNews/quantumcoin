package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const tokenTransportFixtureV1 = "credential-transport-fixture"

func tokenHeadersV1(keys, auth []string) http.Header {
	h := make(http.Header)
	for _, value := range keys {
		h.Add("X-QC-API-KEY", value)
	}
	for _, value := range auth {
		h.Add("Authorization", value)
	}
	return h
}

func TestTokenBindRegressionAmbiguousHeaders(t *testing.T) {
	clearAPITokenTestEnv(t)
	t.Setenv("QC_API_TOKEN", tokenTransportFixtureV1)
	alias := tokenHeadersV1([]string{tokenTransportFixtureV1}, nil)
	alias["x-qc-api-key"] = []string{"wrong"}
	authAlias := tokenHeadersV1(nil, []string{"Bearer " + tokenTransportFixtureV1})
	authAlias["authorization"] = []string{"Bearer wrong"}
	nilKey := tokenHeadersV1(nil, []string{"Bearer " + tokenTransportFixtureV1})
	nilKey[http.CanonicalHeaderKey("X-QC-API-KEY")] = nil
	for _, row := range []struct {
		name    string
		headers http.Header
	}{
		{"duplicate-key", tokenHeadersV1([]string{tokenTransportFixtureV1, "wrong"}, nil)},
		{"duplicate-identical-key", tokenHeadersV1([]string{tokenTransportFixtureV1, tokenTransportFixtureV1}, nil)},
		{"duplicate-bearer", tokenHeadersV1(nil, []string{"Bearer " + tokenTransportFixtureV1, "Bearer wrong"})},
		{"conflicting-pair", tokenHeadersV1([]string{tokenTransportFixtureV1}, []string{"Bearer wrong"})},
		{"key-and-basic", tokenHeadersV1([]string{tokenTransportFixtureV1}, []string{"Basic invalid"})},
		{"empty-key-fallback", tokenHeadersV1([]string{""}, []string{"Bearer " + tokenTransportFixtureV1})},
		{"empty-auth-fallback", tokenHeadersV1([]string{tokenTransportFixtureV1}, []string{""})},
		{"key-case-alias", alias}, {"auth-case-alias", authAlias}, {"nil-key-fallback", nilKey},
	} {
		t.Run(row.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/blocks", nil)
			req.Header = row.headers
			w := httptest.NewRecorder()
			rootAllowed := checkAPIToken(w, req)
			minerAllowed := minerControlHeaderTokenValid(req)
			if rootAllowed || minerAllowed {
				t.Fatalf("ambiguous credentials accepted: root=%v miner=%v", rootAllowed, minerAllowed)
			}
			assertRoutingJSONErrorV1(t, w.Code, w.Header(), w.Body.Bytes(), 401)
		})
	}
}

func TestTokenBindMatchingHeaderForms(t *testing.T) {
	clearAPITokenTestEnv(t)
	t.Setenv("QC_API_TOKEN", tokenTransportFixtureV1)
	for _, headers := range []http.Header{
		tokenHeadersV1([]string{tokenTransportFixtureV1}, nil), tokenHeadersV1(nil, []string{"Bearer " + tokenTransportFixtureV1}),
		tokenHeadersV1(nil, []string{"bEaReR " + tokenTransportFixtureV1}),
		tokenHeadersV1([]string{tokenTransportFixtureV1}, []string{"Bearer " + tokenTransportFixtureV1}),
		{"x-qc-api-key": {tokenTransportFixtureV1}},
	} {
		req := httptest.NewRequest("GET", "/api/blocks", nil)
		req.Header = headers
		if !checkAPIToken(httptest.NewRecorder(), req) || !minerControlHeaderTokenValid(req) {
			t.Fatal("valid header form rejected")
		}
	}
	if checkAPIToken(httptest.NewRecorder(), nil) || minerControlHeaderTokenValid(nil) {
		t.Fatal("nil request authenticated")
	}
}

type tokenTransportBodyV1 struct {
	io.Reader
	reads int
}

func (b *tokenTransportBodyV1) Read(p []byte) (int, error) { b.reads++; return b.Reader.Read(p) }
func (*tokenTransportBodyV1) Close() error                 { return nil }

func TestTokenBindNeverReadsBodyOrQueryForAuth(t *testing.T) {
	clearAPITokenTestEnv(t)
	for _, network := range []string{"local-release", "testnet-public", "mainnet", "unrecognized"} {
		for _, dev := range []string{"0", "1"} {
			for _, bind := range []string{"127.0.0.1:8082", "0.0.0.0:8082"} {
				t.Run(network+"/"+dev+"/"+bind, func(t *testing.T) {
					t.Setenv("QC_NETWORK", network)
					t.Setenv("QC_DEV_MODE", dev)
					t.Setenv("QC_API_LISTEN", bind)
					t.Setenv("QC_API_TOKEN", tokenTransportFixtureV1)
					for _, payload := range []string{"token=" + tokenTransportFixtureV1, `{"token":"` + tokenTransportFixtureV1 + `"}`} {
						body := &tokenTransportBodyV1{Reader: strings.NewReader(payload)}
						req := httptest.NewRequest("POST", "/api/tx/send?token="+tokenTransportFixtureV1, body)
						w := httptest.NewRecorder()
						if checkAPIToken(w, req) || body.reads != 0 {
							t.Fatal("URL/body credential used or read")
						}
						assertRoutingJSONErrorV1(t, w.Code, w.Header(), w.Body.Bytes(), 401)
						req.Header.Set("X-QC-API-KEY", tokenTransportFixtureV1)
						if !checkAPIToken(httptest.NewRecorder(), req) || body.reads != 0 {
							t.Fatal("header credential did not stay independent of body")
						}
					}
					t.Setenv("QC_API_TOKEN", "")
					req := httptest.NewRequest("GET", "/api/blocks", nil)
					req.Header.Set("X-QC-API-KEY", tokenTransportFixtureV1)
					if checkAPIToken(httptest.NewRecorder(), req) {
						t.Fatal("unconfigured token allowed")
					}
				})
			}
		}
	}
}

func TestTokenBindPublicPolicyIgnoresNetworkLabels(t *testing.T) {
	clearCORSTestEnv(t)
	for _, network := range []string{"local-release", "mainnet", "testnet-public", "unrecognized"} {
		t.Setenv("QC_NETWORK", network)
		for _, bind := range []string{"0.0.0.0:8082", ":8082", "[::]:8082", "192.0.2.1:8082", "node.example:8082", "localhost:8082", "invalid"} {
			t.Setenv("QC_API_TOKEN", "")
			if !apiRestrictedBindV1(bind) || validateAPIBindAuthV1(bind) == nil {
				t.Fatalf("exposed bind accepted empty token: %s/%s", network, bind)
			}
			t.Setenv("QC_API_TOKEN", tokenTransportFixtureV1)
			if validateAPIBindAuthV1(bind) != nil {
				t.Fatal("configured exposed policy rejected", bind)
			}
		}
	}
	for _, bind := range []string{"127.0.0.1:8082", "[::1]:8082", "[::ffff:127.0.0.1]:8082"} {
		if apiRestrictedBindV1(bind) {
			t.Fatal("numeric loopback not recognized", bind)
		}
	}
	for _, flag := range []string{"QC_PUBLIC_TESTNET", "QC_API_PUBLIC_TESTNET", "QC_TESTNET_PUBLIC_API"} {
		t.Setenv(flag, "1")
		if !apiRestrictedBindV1("127.0.0.1:8082") {
			t.Fatal("public-testnet flag loosened policy")
		}
		t.Setenv(flag, "")
	}
}

func TestTokenBindWireDuplicateHeadersAndCORS(t *testing.T) {
	externalMiningMux(t)
	t.Setenv("QC_CORS_ALLOWED_ORIGINS", "https://wallet.example")
	h := newHTTPAPIHandlerForListenV1(nil, NewEndpointLimiter(), "0.0.0.0:8082")
	server := httptest.NewServer(h)
	defer server.Close()
	client := &http.Client{Timeout: 3 * time.Second}
	for _, row := range []struct {
		headers http.Header
		origin  string
		status  int
	}{
		{tokenHeadersV1([]string{actualMuxTestTokenV1}, nil), "https://wallet.example", 200},
		{tokenHeadersV1(nil, []string{"Bearer " + actualMuxTestTokenV1}), "", 200},
		{tokenHeadersV1([]string{actualMuxTestTokenV1, "wrong"}, nil), "", 401},
		{tokenHeadersV1(nil, []string{"Bearer " + actualMuxTestTokenV1, "Bearer wrong"}), "", 401},
		{tokenHeadersV1([]string{actualMuxTestTokenV1}, []string{"Bearer wrong"}), "", 401},
		{tokenHeadersV1([]string{actualMuxTestTokenV1}, nil), "https://not-allowed.example", 403},
	} {
		req, err := http.NewRequest("GET", server.URL+"/api/blocks", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header = row.headers
		if row.origin != "" {
			req.Header.Set("Origin", row.origin)
		}
		response, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != row.status {
			t.Fatalf("wire status=%d want=%d", response.StatusCode, row.status)
		}
		if row.status >= 400 {
			assertRoutingJSONErrorV1(t, response.StatusCode, response.Header, body, row.status)
		}
	}
}
