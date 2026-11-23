package apiclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHealthRejectsGenericOrConflictingOK(t *testing.T) {
	for name, payload := range map[string]string{
		"generic ok":          `{"ok":true}`,
		"conflicting success": `{"ok":true,"success":false}`,
		"wrong service":       `{"ok":true,"service":"Other API","healthVersion":"QC_NODE_HEALTH_V1"}`,
		"wrong version":       `{"ok":true,"service":"QuantumCoin API","healthVersion":"other"}`,
		"duplicate field":     `{"ok":false,"ok":true,"service":"QuantumCoin API","healthVersion":"QC_NODE_HEALTH_V1"}`,
		"case alias":          `{"ok":false,"OK":true,"service":"QuantumCoin API","healthVersion":"QC_NODE_HEALTH_V1"}`,
		"trailing document":   `{"ok":true} {"ok":true}`,
		"oversize":            strings.Repeat(" ", 65537),
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, payload)
			}))
			defer server.Close()
			state, err := ProbeHealth(context.Background(), server.URL+"/api/health", "")
			if err == nil || state.Reachable {
				t.Fatal("unrelated or conflicting service accepted as node")
			}
		})
	}
}

func TestHealthDoesNotFollowRedirectsOrLeakToken(t *testing.T) {
	called := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer target.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+CanonicalHealthPathV1, http.StatusTemporaryRedirect)
	}))
	defer origin.Close()
	if _, err := ProbeHealth(context.Background(), origin.URL+CanonicalHealthPathV1, "private-fixture"); err == nil || called {
		t.Fatal("health redirect followed or token forwarded")
	}
}

func TestHealthProbeHonorsDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if state, err := ProbeHealth(ctx, server.URL+CanonicalHealthPathV1, ""); err == nil || state.Reachable {
		t.Fatal("timeout reported healthy")
	}
}
