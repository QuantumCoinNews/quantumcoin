package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"quantumcoin/apiclient"
	"testing"
)

func TestWalletStartupReusesConfiguredNodeWithoutLocalBinary(t *testing.T) {
	t.Setenv("HTTP_PORT", "")
	previous := apiPort
	t.Cleanup(func() { apiPort = previous })
	apiPort = ""
	dir := t.TempDir()
	t.Setenv("QC_NODE_DIR", dir)
	t.Setenv("QC_API_TOKEN", "fixture-token")
	for _, mode := range []string{"ready", "unauthorized", "unrelated"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != apiclient.CanonicalHealthPathV1 || r.Header.Get("X-QC-API-KEY") != "fixture-token" {
					http.Error(w, "bad request", 400)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				switch mode {
				case "ready":
					fmt.Fprintf(w, `{"ok":true,"service":%q,"healthVersion":%q}`, apiclient.HealthServiceV1, apiclient.HealthProtocolV1)
				case "unauthorized":
					w.Header().Set("WWW-Authenticate", apiclient.HealthAuthRealmV1)
					w.WriteHeader(401)
					fmt.Fprint(w, `{"success":false}`)
				default:
					fmt.Fprint(w, `{"ok":true}`)
				}
			}))
			defer server.Close()
			origin, _ := url.Parse(server.URL)
			t.Setenv("QC_HTTP_PORT", origin.Port())
			if got := apiBase(); got != server.URL {
				t.Fatalf("configured API moved: %s", got)
			}
			err := startAPIBackground()
			switch mode {
			case "ready":
				if err != nil {
					t.Fatal(err)
				}
			case "unauthorized":
				if !errors.Is(err, apiclient.ErrHealthUnauthorizedV1) {
					t.Fatal(err)
				}
			default:
				if !errors.Is(err, apiclient.ErrHealthContractV1) {
					t.Fatal(err)
				}
			}
			files, err := os.ReadDir(dir)
			if err != nil || len(files) != 0 {
				t.Fatalf("existing service caused local startup artifacts: %v %v", files, err)
			}
		})
	}
	t.Setenv("QC_HTTP_PORT", "invalid")
	if apiBase() != "" {
		t.Fatal("invalid configured port fell back to another node")
	}
}
